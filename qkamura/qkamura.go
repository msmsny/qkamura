package qkamura

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/cobra"
)

func NewQkamuraCommand() *cobra.Command {
	var (
		location      *string
		stayDates     *[]int
		roomIDs       *[]int
		slackChannel  *string
		slackToken    *string
		qkamuraScheme *string
		qkamuraHost   *string
		slackScheme   *string
		slackHost     *string
		debug         *bool
		flagErrors    []error
	)
	cmds := &cobra.Command{
		Use:           "qkamura",
		Short:         "qkamura find qkamura vacancy rooms and notifies",
		Long:          "qkamura find qkamura vacancy rooms specifying location, stayDates, roomIDs and notifies to slack",
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// 空文字チェックはMarkFlagRequiredでするためここでは空文字チェック以外
			if _, ok := locationIDMap[*location]; !ok && *location != "" {
				return fmt.Errorf("invalid location: %s", *location)
			}
			for _, roomID := range *roomIDs {
				if _, ok := roomIDMap[*location][roomID]; !ok {
					return fmt.Errorf("invalid roomID: %d", roomID)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client := retryablehttp.NewClient()
			client.HTTPClient.Timeout = 10 * time.Second
			client.RetryMax = 2
			client.RetryWaitMin = 3 * time.Second
			client.RetryWaitMax = 10 * time.Second
			httpClient := client.StandardClient()
			qkamuraClient := &qkamuraClient{
				client:    httpClient,
				scheme:    *qkamuraScheme,
				host:      *qkamuraHost,
				location:  *location,
				stayDates: *stayDates,
			}
			slackClient := &slackClient{
				client: httpClient,
				scheme: *slackScheme,
				host:   *slackHost,
			}
			qkamura := &qkamura{
				qkamuraClient: qkamuraClient,
				slackClient:   slackClient,
				location:      *location,
				stayDates:     *stayDates,
				roomIDs:       *roomIDs,
				slackChannel:  *slackChannel,
				slackToken:    *slackToken,
				debug:         *debug,
			}

			return qkamura.run()
		},
	}

	flags := cmds.Flags()
	flags.SortFlags = false
	location = flags.String("location", "tateyama", "qkamura location, e.g.: tateyama, izu")
	stayDates = flags.IntSlice("stay-dates", []int{20210731, 20210807}, "stay dates, e.g.: 20210731,20210807")
	roomIDs = flags.IntSlice("room-ids", []int{1, 7}, `qkamura roomIDs:
tateyama:
	1: 【オーシャンビュー／禁煙／３０㎡】<br>和室１０畳　バス・トイレ・広縁付き
	3: 【オーシャンビュー／禁煙】　洋室ツイン　バス・トイレ付
	4: 【オーシャンビュー／禁煙／３０㎡】<br>洋室ツイン　トイレ付き
	7: 【オーシャンビュー／禁煙／３０㎡】<br>和洋室ツイン　小上がりの座敷・トイレ付き
izu:
	1: 和洋室・禁煙
	2: 和室・禁煙
	5: 洋室・禁煙`)
	slackChannel = flags.String("slack-channel", "", "slack channel to notify")
	slackToken = flags.String("slack-token", "", "slack token to notify")
	qkamuraScheme = flags.String("qkamura-scheme", "https", "qkamura API scheme")
	qkamuraHost = flags.String("qkamura-host", "www.qkamura.or.jp", "qkamura API host")
	slackScheme = flags.String("slack-scheme", "https", "slack API scheme")
	slackHost = flags.String("slack-host", "slack.com", "slack API host")
	debug = flags.Bool("debug", false, "output results instead of slack post")
	flagErrors = append(
		flagErrors,
		cobra.MarkFlagRequired(flags, "slack-channel"),
		cobra.MarkFlagRequired(flags, "slack-token"),
	)

	return cmds
}

var (
	locationIDMap = map[string]int{
		"tateyama": 23260012,
		"izu":      31260022,
	}
	roomIDMap = map[string]map[int]string{
		"tateyama": {
			1: "【オーシャンビュー／禁煙／３０㎡】<br>和室１０畳　バス・トイレ・広縁付き",
			3: "【オーシャンビュー／禁煙】　洋室ツイン　バス・トイレ付",
			4: "【オーシャンビュー／禁煙／３０㎡】<br>洋室ツイン　トイレ付き",
			7: "【オーシャンビュー／禁煙／３０㎡】<br>和洋室ツイン　小上がりの座敷・トイレ付き",
		},
		"izu": {
			1: "和洋室・禁煙",
			2: "和室・禁煙",
			5: "洋室・禁煙",
		},
	}
)

type qkamura struct {
	qkamuraClient *qkamuraClient
	slackClient   *slackClient
	location      string
	stayDates     []int
	roomIDs       []int
	slackChannel  string
	slackToken    string
	debug         bool
}

func (q *qkamura) run() error {
	// オプション指定の日付はYYYYMMDD
	startDate, err := time.Parse("20060102", strconv.Itoa(min(q.stayDates)))
	if err != nil {
		return fmt.Errorf("startDate time.Parse: %s", err)
	}
	endDate, err := time.Parse("20060102", strconv.Itoa(max(q.stayDates)))
	if err != nil {
		return fmt.Errorf("endDate time.Parse: %s", err)
	}
	// 開始日/終了日の範囲指定ですべて取得してレスポンスからstayDatesに該当する日付をピックアップする
	reservation, err := q.qkamuraClient.get(q.location, startDate, endDate)
	if err != nil {
		return fmt.Errorf("qkamuraClient.Get: %s", err)
	}
	postMessages := []string{}

	for _, room := range reservation.Rooms {
		matchRoomID := false
		for _, roomID := range q.roomIDs {
			if roomID == room.RoomID {
				matchRoomID = true
				break
			}
		}
		if !matchRoomID {
			continue
		}

		for _, vacancy := range room.Vacancies {
			matchDate := false
			for _, stayDate := range q.stayDates {
				stayDateTime, err := time.Parse("20060102", strconv.Itoa(stayDate))
				if err != nil {
					return fmt.Errorf("stayDate time.Parse: %s", err)
				}
				if vacancy.Date == stayDateTime.Format("2006/1/2") {
					matchDate = true
					break
				}
			}
			if !matchDate {
				continue
			}
			if vacancy.Count > 0 {
				// ここもmapのチェック省略
				roomDetail := roomIDMap[q.location][room.RoomID]
				postMessages = append(
					postMessages,
					fmt.Sprintf("日付:%s\n部屋タイプ: %s\n室数: %d", vacancy.Date, roomDetail, vacancy.Count),
				)
			}

			fmt.Printf("location: %s, roomID: %d, date: %s, count: %d\n", q.location, room.RoomID, vacancy.Date, vacancy.Count)
		}
	}

	if len(postMessages) > 0 {
		message := fmt.Sprintf("Qkamura vacancy notification\n\nlocation: %s\n%s", q.location, strings.Join(postMessages, "\n"))
		if q.debug {
			fmt.Println(message)
		} else if err := q.slackClient.post(q.slackChannel, q.slackToken, message); err != nil {
			return fmt.Errorf("q.slackClient.post: %s, message: %s", err, message)
		}
	}

	return nil
}

type Reservation struct {
	Rooms []*Room `json:"rooms"`
}

type Room struct {
	Vacancies []*Vacancy `json:"aki"`
	RoomID    int        `json:"room_id,string"`
}

type Vacancy struct {
	Count int    `json:"aki_num,string"`
	Date  string `json:"aki_date"` // 2006/1/2
}

type PostMessage struct {
	Channel string `json:"channel"`
	Message string `json:"text"`
}

func min(ns []int) int {
	min := ns[0]
	for _, n := range ns {
		min = int(math.Min(float64(min), float64(n)))
	}

	return min
}

func max(ns []int) int {
	max := ns[0]
	for _, n := range ns {
		max = int(math.Max(float64(max), float64(n)))
	}

	return max
}

type qkamuraClient struct {
	client    *http.Client
	scheme    string
	host      string
	location  string
	stayDates []int
}

// ID, 開始日, 終了日を指定して空室検索APIを叩きレスポンスを取得する
// レスポンスにはroomID, 日付ごとの空室数が入っている
func (q *qkamuraClient) get(location string, startDate, endDate time.Time) (*Reservation, error) {
	// locationはバリデーション済なのでチェックは省略
	resp, err := q.client.Get(q.buildURL(locationIDMap[location], startDate, endDate))
	if err != nil {
		return nil, fmt.Errorf("client.Get: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		rawBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("qkamuraClient response is %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("qkamuraClient response is %d: %s", resp.StatusCode, rawBody)
	}

	// bodyを加工してJSONを取り出す
	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %s", err)
	}
	body := string(rawBody)
	body = strings.Replace(body, "getStockData(", "", 1)
	body = strings.Replace(body, ")", "", 1)
	bodyJSON := strings.Replace(body, "'", `"`, -1)

	reservation := &Reservation{}
	if err := json.Unmarshal([]byte(bodyJSON), reservation); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %s", err)
	}

	return reservation, nil
}

// 最新のURLの例
// https://www.qkamura.or.jp/qkamura/api/ypro/v2/ypro_stocksearch_api.asp?id=23260012&roomId=all&startDate=2021/4/25&endDate=2021/6/5&input_data=index=0,id=23260012,kid=00027,room=all_none,start_date=2021/4/25,end_date=2021/6/5,div=con_search_month_fix,name=%E4%BC%91%E6%9A%87%E6%9D%91%E9%A4%A8%E5%B1%B1,cal_number=1,room_position=0&callback=jQuery2140524766539026686_1619428265073&_=1619428265075
// id: ロケーション固有の数値, 23260012がtateyama
// roomId: allで指定なし, all_noneにすると先頭の部屋IDしか取れないので注意
// planIdのパラメータは不要
func (q *qkamuraClient) buildURL(id int, startDate, endDate time.Time) string {
	return fmt.Sprintf(
		"%s://%s/qkamura/api/ypro/v2/ypro_stocksearch_api.asp?id=%d&roomId=all&startDate=%s&endDate=%s",
		q.scheme,
		q.host,
		id,
		// APIに指定する日付はYYYY/M/D
		startDate.Format("2006/1/2"),
		endDate.Format("2006/1/2"),
	)
}

type slackClient struct {
	client *http.Client
	scheme string
	host   string
}

func (s *slackClient) post(channel, token, message string) error {
	messageBodyParams := &PostMessage{
		Channel: channel,
		Message: message,
	}
	messageBody, err := json.Marshal(messageBodyParams)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err)
	}
	url := fmt.Sprintf("%s://%s/api/chat.postMessage", s.scheme, s.host)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageBody))
	if err != nil {
		return fmt.Errorf("http.NewRequest: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("s.client.Do: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		rawBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("slackClient response is %d", resp.StatusCode)
		}
		return fmt.Errorf("slackClient response is %d: %s", resp.StatusCode, rawBody)
	}

	return nil
}
