# qkamura

[![Go Report Card](https://goreportcard.com/badge/github.com/msmsny/qkamura)](https://goreportcard.com/report/github.com/msmsny/qkamura)
[![Test](https://github.com/msmsny/qkamura/actions/workflows/test.yml/badge.svg)](https://github.com/msmsny/qkamura/actions/workflows/test.yml)
[![Coverage Status](https://coveralls.io/repos/github/msmsny/qkamura/badge.svg?branch=master)](https://coveralls.io/github/msmsny/qkamura?branch=master)

Qkamura vacancy notifier

## Install

```bash
$ go get github.com/msmsny/qkamura
```

## Usage

```bash
$ qkamura --help
qkamura find qkamura vacancy rooms specifying location, stayDates, roomIDs and notifies to slack

Usage:
  qkamura [flags]

Flags:
      --location string         qkamura location, e.g.: tateyama, izu (default "tateyama")
      --stay-dates ints         stay dates, e.g.: 20210731,20210807
      --room-ids ints           qkamura roomIDs:
                                tateyama:
                                	1: 【オーシャンビュー／禁煙／３０㎡】<br>和室１０畳　バス・トイレ・広縁付き
                                	3: 【オーシャンビュー／禁煙】　洋室ツイン　バス・トイレ付
                                	4: 【オーシャンビュー／禁煙／３０㎡】<br>洋室ツイン　トイレ付き
                                	7: 【オーシャンビュー／禁煙／３０㎡】<br>和洋室ツイン　小上がりの座敷・トイレ付き
                                izu:
                                	1: 和洋室・禁煙
                                	2: 和室・禁煙
                                	5: 洋室・禁煙
      --slack-channel string    slack channel to notify
      --slack-token string      slack token to notify
      --qkamura-scheme string   qkamura API scheme (default "https")
      --qkamura-host string     qkamura API host (default "www.qkamura.or.jp")
      --slack-scheme string     slack API scheme (default "https")
      --slack-host string       slack API host (default "slack.com")
      --debug                   output results instead of slack post
  -h, --help                    help for qkamura
```

With options

```bash
$ qkamura \
  --location tateyama \
  --stay-dates 20210731,20210806 \
  --room-ids 1,7 \
  --slack-channel your-channel \
  --slack-token xxxxx
```

![qkamura_notifier](https://user-images.githubusercontent.com/1556298/118208237-58f33400-b4a1-11eb-8602-73f18b9bb606.png)
