package main

import (
  "net/http"
  "net/url"

  "database/sql"
  _ "github.com/go-sql-driver/mysql"

	"fmt"
	"log"
  "strconv"
  "strings"

  "github.com/ikawaha/slackbot"
)

var (
  APITOKEN = "##"
  ch = "#general"
)

type article_data struct {
  title       string
  user_name   string
}

func db_connect() *sql.DB {
  db,err := sql.Open("mysql","###")

  if err != nil {
    panic(err.Error())
  }
  return db
}

func connect() *slackbot.Bot{
	bot, err := slackbot.New(APITOKEN)
	if err != nil {
		log.Fatal(err)
	}
	return bot
}

func extract_from_db(db *sql.DB, query string) []string {
	rows, err := db.Query(query)

	var data_extracted_from_db []string
	//データが抽出できているかのエラー検出
	if err != nil {
		data_extracted_from_db = append(data_extracted_from_db, "false")
		fmt.Println(err)
		// return data_extracted_from_db
	}
	colum, err := rows.Columns()

	values := make([]sql.RawBytes, len(colum))
	scanArgs := make([]interface{}, len(values))

	for i := range values {
		scanArgs[i] = &values[i]
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		for _, col := range values {
			//データが取得できているかのエラー検出
			if col == nil {
				data_extracted_from_db = append(data_extracted_from_db, "false")
			} else {
				data_extracted_from_db = append(data_extracted_from_db, string(col))
			}
		}
	}
  return data_extracted_from_db
}

// 記事の件数によって新規投稿か更新かを判定
func count_article_from_db(db *sql.DB, num_now int,bot *slackbot.Bot, finish_update []int) (int,[]int) {
	//sqlクエリ
	query := "select count(*) from articles"

  num_articles , _ := strconv.Atoi(extract_from_db(db,query)[0])
  if num_articles < num_now {
    return num_articles,finish_update
  }

  // 記事件数が同じなら記事が更新されているかを確認する
  if num_articles == num_now {
    finish_update = update_check(db, bot, finish_update)
    return num_now,finish_update
  }

  // 記事件数が増えていたら新規投稿と判定する
  query = "select id, user_id, title, published from articles order by id desc limit 1"
  newest_article := extract_from_db(db, query)
  if strings.Index(newest_article[2], "[WIP]") == -1 {
    // 公開状況の確認
    flg, _ := strconv.Atoi(newest_article[3])
    fmt.Println(flg)
    if flg != 0 {
      echoo(bot, newest_article[1], newest_article[2], newest_article[0])
    }
  }
  return num_articles,finish_update
}

func update_check(db *sql.DB, bot *slackbot.Bot, finish_update []int) []int{
  query := "select id, article_id, new_title, old_title from update_histories order by id desc limit 1"
  data  := extract_from_db(db, query)
  bo , _ := strconv.Atoi(data[0])
  if (bo != finish_update[len(finish_update)-1]) && (strings.Index(data[2], "[WIP]") == -1) && (strings.Index(data[3], "[WIP]") == 0){
    q := "select published from articles where id=" + data[1]
    if extract_from_db(db, q)[0] == "1" {
      echoo(bot, "0", data[2], data[1])
      finish_update = append(finish_update, bo)
    }
  }
  return finish_update
}

func echoo(b *slackbot.Bot,user_id string,title string, articles_id string) {
  channel := "https://slack.com/api/chat.postMessage"

  push_text := title + "\nがRe:techに投稿されました"
  push_text += "\nhttp://dryer.wsl.mind.meiji.ac.jp:3000/articles/" + articles_id

  resp, _ := http.PostForm(
        channel,
        url.Values{
          "token": {APITOKEN},
          "channel":{ch},
          "text":{push_text},
          "as_user":{"true"},
      },
    )
  defer resp.Body.Close()
}

func main() {
  db := db_connect()
  bot := connect()
  num_now , _:= strconv.Atoi(extract_from_db(db, "select count(*) from articles")[0])
  latest , _ := strconv.Atoi(extract_from_db(db, "select id from update_histories order by id desc limit 1")[0])
  finish_update := []int{latest}
  for {
    num_now,finish_update = count_article_from_db(db, num_now, bot, finish_update)
  }
}
