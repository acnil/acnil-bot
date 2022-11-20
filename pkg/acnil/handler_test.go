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
			member acnil.Member
		)

		BeforeEach(func() {
			member = acnil.Member{
				Nickname:    "MetalBlueberry",
				TelegramID:  "12345",
				Permissions: "yes",
			}
		})
		Describe("When /start message is received", func() {
			BeforeEach(func() {
				sender := &tele.User{
					ID:        12345,
					FirstName: "Victor",
					LastName:  "Perez",
					Username:  "MetalBlueberry",
				}
				text := "/start"
				mockTeleContext.EXPECT().Sender().Return(sender).AnyTimes()
				mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Text:   "/start",
				}).AnyTimes()
			})
			It("Should reply with hello world message", func() {
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Bienvenido al bot de Acnil"))
					return nil
				})
				err := h.Start(mockTeleContext, member)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

})
