package acnil_test

import (
	"github.com/golang/mock/gomock"
	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/acnil/mock_acnil"
	"github.com/metalblueberry/acnil-bot/pkg/acnil/mock_telebot_v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tele "gopkg.in/telebot.v3"
)

//go:generate mockgen -source=handler.go -destination mock_acnil/mock.go
var _ = Describe("Handler", func() {

	var (
		ctrl                *gomock.Controller
		mockMembersDatabase *mock_acnil.MockMembersDatabase
		mockGameDatabase    *mock_acnil.MockGameDatabase
		mockTeleContext     *mock_telebot_v3.MockContext
		h                   acnil.Handler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockMembersDatabase = mock_acnil.NewMockMembersDatabase(ctrl)
		mockGameDatabase = mock_acnil.NewMockGameDatabase(ctrl)
		mockTeleContext = mock_telebot_v3.NewMockContext(ctrl)

		h = acnil.Handler{
			MembersDB: mockMembersDatabase,
			GameDB:    mockGameDatabase,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("An authorised member", func() {
		var (
			member *acnil.Member
			sender *tele.User
		)

		BeforeEach(func() {
			member = &acnil.Member{
				Nickname:    "MetalBlueberry",
				TelegramID:  "12345",
				Permissions: "si",
			}
			mockMembersDatabase.EXPECT().Get(gomock.Any(), member.TelegramIDInt()).Return(member, nil)
			sender = &tele.User{
				ID:        12345,
				FirstName: "Victor",
				LastName:  "Perez",
				Username:  "MetalBlueberry",
			}
			mockTeleContext.EXPECT().Sender().Return(sender).AnyTimes()

		})

		Describe("When /start message is received", func() {
			BeforeEach(func() {
				text := "/start"
				mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Text:   text,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()
			})
			It("Should reply with hello world message", func() {
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Bienvenido al bot de Acnil"))
					return nil
				})
				err := h.Start(mockTeleContext)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Describe("With a board game library", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Find(gomock.Any(), "Game1").Return([]acnil.Game{
					{
						ID:   "1",
						Name: "Game1",
					},
				}, nil)
			})

			Describe("When Text is sent", func() {
				BeforeEach(func() {
					text := "Game1"
					mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
					mockTeleContext.EXPECT().Message().Return(&tele.Message{
						Sender: sender,
						Text:   text,
						Chat: &tele.Chat{
							Type: tele.ChatPrivate,
						},
					}).AnyTimes()
				})
				It("Should reply with game details", func() {
					mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
						Expect(sent).To(ContainSubstring("Game1"))
						return nil
					})
					err := h.OnText(mockTeleContext)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})

})
