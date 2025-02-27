package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

type App struct {
	channelSecret string
	bot *messaging_api.MessagingApiAPI
}

func NewApp(channelSecret, channelToken string) (*App, error) {
	bot, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		return nil, err
	}
	return &App{
		channelSecret: channelSecret,
		bot: bot,
	}, nil
}

func (app *App) HandleEvents(c *gin.Context) {
	req, err := webhook.ParseRequest(app.channelSecret, c.Request)
	if err != nil {
		log.Println("Error parsing request:", err)
		c.Status(http.StatusBadRequest)
		return
	}

	log.Println("Handling events...")

	for _, event := range req.Events {
		log.Printf("/linebot called %+v...\n", event)

		switch e := event.(type) {
		case webhook.MessageEvent:
			switch message := e.Message.(type) {
			case webhook.FollowEvent:
				_, err := app.bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
						Messages: []messaging_api.MessageInterface{
							&messaging_api.TextMessage{
								Text: "สวัสดีครับ! หากคุณนึกไม่ออกว่าจะดูอะไร เราจะช่วยคุณเอง!",
								QuickReply: &messaging_api.QuickReply{
									Items: []messaging_api.QuickReplyItem{
										{
											Action: &messaging_api.MessageAction{
												Label: "สุ่มหนัง",
												Text:  "สุ่มหนัง",
											},
										},
										{
											Action: &messaging_api.MessageAction{
												Label: "สุ่มหนังรัก",
												Text:  "สุ่มหนังรัก",
											},
										},
										{
											Action: &messaging_api.MessageAction{
												Label: "สุ่มหนังตลก",
												Text:  "สุ่มหนังตลก",
											},
										},
										{
											Action: &messaging_api.MessageAction{
												Label: "สุ่มหนังผี",
												Text:  "สุ่มหนังผี",
											},
										},
									},
								},
							},
						},
					},
				)
				if err != nil {
					log.Println("Error sending greeting message:", err)
				}
			case webhook.TextMessageContent:
				if err := app.handleText(&message, e.ReplyToken); err != nil {
					log.Println(err)
				}
			default:
				log.Printf("Unknown event: %v", event)
			}
		}
	}

	c.Status(http.StatusOK)
}

func (app *App) sendMovieTemplate(replyToken string, randomMovie Movie) error {
	imageURL := "https://image.tmdb.org/t/p/w500/" + randomMovie.Poster

	_, err := app.bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				&messaging_api.TemplateMessage{
					AltText: "ส่งหนังให้คุณ",
					Template: &messaging_api.CarouselTemplate{
						Columns: []messaging_api.CarouselColumn{
							{
								ThumbnailImageUrl: imageURL,
								Title: randomMovie.Title,
								Text: "  ",
								Actions: []messaging_api.ActionInterface{
									&messaging_api.MessageAction{
										Label: "เรื่องย่อ",
										Text:  "ขอเรื่องย่อหน่อย",
									},
								},
							},
							{
								ThumbnailImageUrl: imageURL,
								Title: randomMovie.Title,
								Text: "  ",
								Actions: []messaging_api.ActionInterface{
									&messaging_api.MessageAction{
										Label: "เรื่องย่อ",
										Text:  "ขอเรื่องย่อหน่อย",
									},
								},
							},
						},
					},
				},
				&messaging_api.TextMessage{
					Text: "หวังว่าคุณจะชอบนะ",
					QuickReply: &messaging_api.QuickReply{
						Items: []messaging_api.QuickReplyItem{
							{
								Action: &messaging_api.MessageAction{
									Label: "สุ่มหนัง",
									Text:  "สุ่มหนัง",
								},
							},
							{
								Action: &messaging_api.MessageAction{
									Label: "สุ่มหนังรัก",
									Text:  "สุ่มหนังรัก",
								},
							},
							{
								Action: &messaging_api.MessageAction{
									Label: "สุ่มหนังตลก",
									Text:  "สุ่มหนังตลก",
								},
							},
							{
								Action: &messaging_api.MessageAction{
									Label: "สุ่มหนังผี",
									Text:  "สุ่มหนังผี",
								},
							},
						},
					},
				},
			},
		},
	)
	return err
}

var lastMovie Movie // ตัวแปรเก็บหนังที่สุ่มล่าสุด

func (app *App) handleText(message *webhook.TextMessageContent, replyToken string) error {
	var responseText string
	var genre string

	switch message.Text {
	case "สุ่มหนัง":
		genre = "all"
	case "สุ่มหนังรัก":
		genre = "romance"
	case "สุ่มหนังตลก":
		genre = "comedy"
	case "สุ่มหนังผี":
		genre = "horror"
	case "ขอเรื่องย่อหน่อย":
		if lastMovie.Title == "" {
			responseText = "ยังไม่มีหนังที่สุ่มมา"
		} else {
			responseText = "เรื่องย่อ: " + lastMovie.Overview
		}
	default:
		responseText = message.Text
	}

	if genre != "" {
		randomMovie, err := app.fetchRandomMovie(genre)
		if err != nil {
			log.Println("Error fetching random movie:", err)
			responseText = "เกิดข้อผิดพลาดไม่สามารถดึงข้อมูลหนังได้ ลองใหม่อีกครั้ง"
		} else {
			// เก็บหนังที่สุ่มล่าสุด
			lastMovie = randomMovie
			err = app.sendMovieTemplate(replyToken, randomMovie)
			if err != nil {
				log.Println("Error sending movie template:", err)
				responseText = "เกิดข้อผิดพลาดในการส่งข้อความ"
			}
		}
	}

	if responseText != "" {
		if _, err := app.bot.ReplyMessage(
			&messaging_api.ReplyMessageRequest{
				ReplyToken: replyToken,
				Messages: []messaging_api.MessageInterface{
					&messaging_api.TextMessage{
						Text: responseText,
						QuickReply: &messaging_api.QuickReply{
							Items: []messaging_api.QuickReplyItem{
								{
									Action: &messaging_api.MessageAction{
										Label: "สุ่มหนัง",
										Text:  "สุ่มหนัง",
									},
								},
								{
									Action: &messaging_api.MessageAction{
										Label: "สุ่มหนังรัก",
										Text:  "สุ่มหนังรัก",
									},
								},
								{
									Action: &messaging_api.MessageAction{
										Label: "สุ่มหนังตลก",
										Text:  "สุ่มหนังตลก",
									},
								},
								{
									Action: &messaging_api.MessageAction{
										Label: "สุ่มหนังผี",
										Text:  "สุ่มหนังผี",
									},
								},
							},
						},
					},
				},
			},
		); err != nil {
			return err
		}
	}
	return nil
}

type Movie struct {
	Title  string `json:"original_title"`
	Poster string `json:"poster_path"`
	Overview string `json:"overview"`
}

type MovieResponse struct {
	Results []Movie `json:"results"`
}

func (app *App) fetchRandomMovie(genre string) (Movie, error) {
	apiKey := os.Getenv("TMDB_API_KEY")
	var url string

	switch genre {
	case "romance": // หนังรัก
		url = "https://api.themoviedb.org/3/discover/movie?api_key=" + apiKey + "&with_genres=10749&language=th-TH"
	case "comedy": // หนังตลก
		url = "https://api.themoviedb.org/3/discover/movie?api_key=" + apiKey + "&with_genres=35&language=th-TH"
	case "horror": // หนังผี
		url = "https://api.themoviedb.org/3/discover/movie?api_key=" + apiKey + "&with_genres=27&language=th-TH"
	case "all": // ทั้งหมด (popular)
		url = "https://api.themoviedb.org/3/movie/popular?api_key=" + apiKey + "&language=th-TH"
	default:
		return Movie{}, errors.New("genre not recognized")
	}

	resp, err := http.Get(url)
	if err != nil {
		return Movie{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Movie{}, err
	}

	var result struct {
		Results []Movie `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return Movie{}, err
	}

	if len(result.Results) == 0 {
		return Movie{}, errors.New("no movies found")
	}

	// สุ่มหนังจากหนังที่ได้มา
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(result.Results))
	return result.Results[randomIndex], nil
}


func main() {

	app, err := NewApp(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	router.POST("/linebot", app.HandleEvents)

	port := "8080"
	fmt.Println("Server running at http://localhost:" + port + "/")
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}

}
