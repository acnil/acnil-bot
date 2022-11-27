package acnil_test

import (
	"context"

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
		mockSender          *mock_acnil.MockSender
		mockTeleContext     *mock_telebot_v3.MockContext
		h                   acnil.Handler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockMembersDatabase = mock_acnil.NewMockMembersDatabase(ctrl)
		mockGameDatabase = mock_acnil.NewMockGameDatabase(ctrl)
		mockSender = mock_acnil.NewMockSender(ctrl)
		mockTeleContext = mock_telebot_v3.NewMockContext(ctrl)

		h = acnil.Handler{
			MembersDB: mockMembersDatabase,
			GameDB:    mockGameDatabase,
			Bot:       mockSender,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("A new member", func() {
		var (
			newMember *acnil.Member
			admin     *acnil.Member
			sender    *tele.User
		)
		BeforeEach(func() {
			admin = &acnil.Member{
				Nickname:    "MetalBlueberry",
				TelegramID:  "12345",
				Permissions: acnil.PermissionAdmin,
			}
			sender = &tele.User{
				ID:        1,
				FirstName: "New",
				LastName:  "User",
				Username:  "NewUser",
			}
			newMember = &acnil.Member{
				TelegramID:  "1",
				Nickname:    "New User",
				Permissions: "no",
			}
			mockMembersDatabase.EXPECT().Get(gomock.Any(), sender.ID).Return(nil, nil)
			mockMembersDatabase.EXPECT().Append(gomock.Any(), gomock.AssignableToTypeOf(acnil.Member{})).Return(nil).Do(func(_ context.Context, member acnil.Member) {
				Expect(member.Nickname).To(Equal(newMember.Nickname))
				Expect(member.TelegramID).To(Equal(newMember.TelegramID))
				Expect(member.Permissions).To(Equal(acnil.PermissionNo))
			})
			mockMembersDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Member{
				*admin,
				*newMember,
			}, nil)
			mockSender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(r tele.Recipient, msg interface{}, opts ...interface{}) {
				Expect(r.Recipient()).To(Equal(admin.TelegramID))
			})

			mockTeleContext.EXPECT().Sender().Return(sender).AnyTimes()
		})
		Describe("calls /start", func() {
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
			It("Should notify admins and register the user in the table", func() {
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Hola,"))
					return nil
				})
				err := h.Start(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
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
				Permissions: acnil.PermissionYes,
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
			It("Should reply with welcome message message", func() {
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Bienvenido al bot de Acnil"))
					return nil
				})
				err := h.Start(mockTeleContext)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Describe("When Text is sent", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Find(gomock.Any(), "Game1").Return([]acnil.Game{
					{
						ID:   "1",
						Name: "Game1",
					},
				}, nil)
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
		Describe("When Text is an ID", func() {
			BeforeEach(func() {
				text := "1"
				mockGameDatabase.EXPECT().Get(gomock.Any(), text, "").Return(&acnil.Game{
					ID:   "1",
					Name: "Game1",
				}, nil)
				mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Text:   text,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()
			})
			It("Should reply with game details by ID", func() {
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					return nil
				})
				err := h.OnText(mockTeleContext)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Describe("When an user attempts to take a game that is available", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
					ID:   "1",
					Name: "Game1",
				}, nil)

				mockTeleContext.EXPECT().Data().Return(acnil.Game{
					ID:   "1",
					Name: "Game1",
				}.Data()).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()
				mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{
					ID:     "1",
					Name:   "Game1",
					Holder: member.Nickname,
				}))
			})
			It("must allow the user to take the game", func() {
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					return nil
				})
				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnTake(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
		Describe("When an user attempts to take a game that is NOT available", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
					ID:     "1",
					Name:   "Game1",
					Holder: "Other Person",
				}, nil)

				mockTeleContext.EXPECT().Data().Return(acnil.Game{
					ID:     "1",
					Name:   "Game1",
					Holder: "Other Person",
				}.Data()).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()
			})
			It("must not change the game ownership and send updated data", func() {
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("te envío los últimos actualizados"))
					return nil
				})
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					return nil
				})

				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnTake(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
		Describe("When an user attempts to take a game that doesn't exist", func() {
			BeforeEach(func() {
				mockTeleContext.EXPECT().Data().Return(acnil.Game{
					ID:     "1",
					Name:   "Game1",
					Holder: "Other Person",
				}.Data()).AnyTimes()
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(nil, nil)
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()
			})
			It("must notify the user by editing the message", func() {
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("No he podido encontrar el juego"))
					return nil
				})

				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnTake(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
		Describe("When an user returns a game that is owned by himself", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
					ID:     "1",
					Name:   "Game1",
					Holder: member.Nickname,
				}, nil)

				mockTeleContext.EXPECT().Data().Return(acnil.Game{
					ID:   "1",
					Name: "Game1",
				}.Data()).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()
				mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{
					ID:   "1",
					Name: "Game1",
				}))
			})
			It("the game must be updated with empty holder", func() {
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					return nil
				})
				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnReturn(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
		Describe("When an user returns a game that is owned not owned by himself", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
					ID:     "1",
					Name:   "Game1",
					Holder: "Other User",
				}, nil)

				mockTeleContext.EXPECT().Data().Return(acnil.Game{
					ID:   "1",
					Name: "Game1",
				}.Data()).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()

			})
			It("the game must not be updated and new data must be returned", func() {
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("te envío los últimos actualizados"))
					return nil
				})
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					return nil
				})
				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnReturn(mockTeleContext)
				Expect(err).To(BeNil())
			})

		})
	})

})
