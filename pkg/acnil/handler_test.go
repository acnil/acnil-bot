package acnil_test

import (
	"context"
	"time"

	"github.com/acnil/acnil-bot/pkg/acnil"
	. "github.com/acnil/acnil-bot/pkg/acnil/matchers"
	"github.com/acnil/acnil-bot/pkg/acnil/mock_acnil"
	"github.com/acnil/acnil-bot/pkg/acnil/mock_telebot_v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	tele "gopkg.in/telebot.v3"
)

var (
	redDot   = "🔴"
	greenDot = "🟢"
)

//go:generate mockgen -source=handler.go -destination mock_acnil/mock.go
//go:generate mockgen -destination mock_telebot_v3/mock.go gopkg.in/telebot.v3 Context
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
				Nickname:         "MetalBlueberry",
				TelegramID:       "12345",
				Permissions:      acnil.PermissionAdmin,
				TelegramName:     "Victor Perez",
				TelegramUsername: "MetalBlueberry",
			}
			sender = &tele.User{
				ID:        1,
				FirstName: "New",
				LastName:  "User",
			}
			newMember = &acnil.Member{
				TelegramID:       "1",
				Nickname:         "New User",
				Permissions:      "no",
				TelegramName:     "New User",
				TelegramUsername: "Username",
			}
			mockTeleContext.EXPECT().Sender().Return(sender).AnyTimes()
		})
		Describe("calls /start for the first time", func() {
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

				mockMembersDatabase.EXPECT().Get(gomock.Any(), sender.ID).Do(func(context.Context, int64) {
					// List should only be called after Get is called
					mockMembersDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Member{
						*admin,
						*newMember,
					}, nil)
				}).Return(nil, nil)

			})
			It("Should notify admins and register the user in the table", func() {
				mockSender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(r tele.Recipient, msg interface{}, opts ...interface{}) {
					Expect(r.Recipient()).To(Equal(admin.TelegramID))
					Expect(msg).To(ContainSubstring(newMember.Nickname))
				})

				mockMembersDatabase.EXPECT().Append(gomock.Any(), gomock.AssignableToTypeOf(acnil.Member{})).Return(nil).Do(func(_ context.Context, member acnil.Member) {
					Expect(member.Nickname).To(Equal(newMember.Nickname))
					Expect(member.TelegramID).To(Equal(newMember.TelegramID))
					Expect(member.Permissions).To(Equal(acnil.PermissionNo))
				})

				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Hola,"))
					return nil
				})
				err := h.Start(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
	})
	Describe("A new member with username", func() {
		var (
			newMember *acnil.Member
			admin     *acnil.Member
			sender    *tele.User
		)
		BeforeEach(func() {
			admin = &acnil.Member{
				Nickname:         "MetalBlueberry",
				TelegramID:       "12345",
				Permissions:      acnil.PermissionAdmin,
				TelegramName:     "Victor Perez",
				TelegramUsername: "MetalBlueberry",
			}
			sender = &tele.User{
				ID:        1,
				FirstName: "New",
				LastName:  "User",
				Username:  "NewUsername",
			}
			newMember = &acnil.Member{
				TelegramID:       "1",
				Nickname:         "NewUsername",
				Permissions:      "no",
				TelegramName:     "New User",
				TelegramUsername: "Username",
			}
			mockTeleContext.EXPECT().Sender().Return(sender).AnyTimes()
		})
		Describe("calls /start for the first time", func() {
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

				mockMembersDatabase.EXPECT().Get(gomock.Any(), sender.ID).Do(func(context.Context, int64) {
					// List should only be called after Get is called
					mockMembersDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Member{
						*admin,
						*newMember,
					}, nil)
				}).Return(nil, nil)

			})
			It("Should notify admins and register the user in the table", func() {
				mockSender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(r tele.Recipient, msg interface{}, opts ...interface{}) {
					Expect(r.Recipient()).To(Equal(admin.TelegramID))
					Expect(msg).To(ContainSubstring(newMember.Nickname))
				})

				mockMembersDatabase.EXPECT().Append(gomock.Any(), gomock.AssignableToTypeOf(acnil.Member{})).Return(nil).Do(func(_ context.Context, member acnil.Member) {
					Expect(member.Nickname).To(Equal(newMember.Nickname))
					Expect(member.TelegramID).To(Equal(newMember.TelegramID))
					Expect(member.Permissions).To(Equal(acnil.PermissionNo))
				})

				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Hola,"))
					return nil
				})
				err := h.Start(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
	})
	Describe("An administrator ", func() {
		var (
			admin  *acnil.Member
			sender *tele.User
		)
		BeforeEach(func() {
			admin = &acnil.Member{
				Nickname:         "MetalBlueberry",
				TelegramID:       "12345",
				Permissions:      acnil.PermissionAdmin,
				TelegramName:     "Victor Perez",
				TelegramUsername: "MetalBlueberry",
			}
			mockMembersDatabase.EXPECT().Get(gomock.Any(), admin.TelegramIDInt()).Return(admin, nil)
			sender = &tele.User{
				ID:        12345,
				FirstName: "Victor",
				LastName:  "Perez",
				Username:  "MetalBlueberry",
			}
			mockTeleContext.EXPECT().Sender().Return(sender).AnyTimes()
		})
		Describe("Authorises a new user to use the bot", func() {
			var (
				newMember *acnil.Member
			)
			BeforeEach(func() {
				newMember = &acnil.Member{
					TelegramID:       "1",
					Nickname:         "New User",
					Permissions:      "no",
					TelegramName:     "New User",
					TelegramUsername: "Username",
				}
				text := "/fauthorise"
				mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Text:   text,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
				}).AnyTimes()
				mockTeleContext.EXPECT().Data().Return(newMember.TelegramID)

				mockMembersDatabase.EXPECT().Get(gomock.Any(), newMember.TelegramIDInt()).Return(newMember, nil)

			})

			It("Should update excel and notify user about the granted access", func() {
				mockMembersDatabase.EXPECT().Update(gomock.Any(), gomock.Any()).Do(func(_ context.Context, newMemberUpdate acnil.Member) {
					Expect(newMemberUpdate.TelegramID).To(Equal(newMember.TelegramID))
					Expect(newMemberUpdate.Permissions).To(Equal(acnil.PermissionYes))
				})
				mockSender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(r tele.Recipient, msg interface{}, opts ...interface{}) {
					Expect(r.Recipient()).To(Equal(newMember.TelegramID))
					Expect(msg).To(ContainSubstring("Ya tienes acceso!"))
				})
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).Do(func(msg string, any ...interface{}) {
					Expect(msg).To(ContainSubstring("Se ha dado acceso al usuario"))
					Expect(msg).To(ContainSubstring(newMember.Nickname))
				}).Return(nil)

				err := h.OnAuthorise(mockTeleContext)
				Expect(err).To(BeNil())
			})

		})
		Describe("request a list of forgotten games", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
					// Must not be returned if return date is in the future
					{
						// Game due date in the future
						ID:         "1",
						Name:       "Game1",
						Holder:     "Other User",
						ReturnDate: time.Now().Add(time.Hour * 24 * 30),
					},
					// Must not be returned if has no holder/return date
					{
						// Game without due date
						// Game without holder
						ID:   "2",
						Name: "Game2",
					},
					{
						// Game due date in the past
						ID:         "3",
						Name:       "Game3",
						Holder:     "Other User",
						ReturnDate: time.Now().Add(-time.Hour * 24 * 30),
					},
					// Must not be returned if has no return date
					{
						// Game without due date
						ID:     "4",
						Name:   "Game4",
						Holder: "Manual input user",
					},
					// Must not be returned if has no holder
					{
						// Game with due date
						// game without holder
						ID:         "4",
						Name:       "Game4",
						ReturnDate: time.Now().Add(-time.Hour * 24 * 30),
					},
				}, nil)
			})

			It("returns the games that return date is less than today", func() {
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game3"))
					return nil
				}).Times(1)
				err := h.OnForgotten(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
		Describe("Attempts to extend lease a game owned by other user", func() {
			var (
				game *acnil.Game
			)
			BeforeEach(func() {
				game = &acnil.Game{
					ID:         "1",
					Name:       "Game1",
					Holder:     "Other User",
					TakeDate:   time.Now().Add(-21 * 24 * time.Hour),
					ReturnDate: time.Now(),
				}
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(game, nil)
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: game.Card(),
				}).AnyTimes()
			})
			It("Must update the return date", func() {
				mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{})).Do(func(ctx context.Context, g acnil.Game) {
					Expect(g.ReturnDate).To(BeTemporally(">", game.ReturnDate))
				})
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).Do(func(msg string, any ...interface{}) {
					Expect(msg).To(ContainSubstring(game.Name))
				}).Return(nil)
				mockTeleContext.EXPECT().Respond(gomock.Any()).Times(1)

				err := h.OnExtendLease(mockTeleContext)
				Expect(err).To(BeNil())
			})
			Describe("that doesn't have a take date", func() {
				BeforeEach(func() {
					game.TakeDate = time.Time{}
				})

				It("Must inform of the error", func() {
					mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).Do(func(msg string, any ...interface{}) {
						Expect(msg).To(ContainSubstring("Necesito la fecha de prestamos para poder añadir mas dias"))
					}).Return(nil)
					mockTeleContext.EXPECT().Respond(gomock.Any()).Times(1)

					err := h.OnExtendLease(mockTeleContext)
					Expect(err).To(BeNil())
				})
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
				Nickname:         "MetalBlueberry",
				TelegramID:       "12345",
				Permissions:      acnil.PermissionYes,
				TelegramName:     "Victor Perez",
				TelegramUsername: "MetalBlueberry",
			}
			mockMembersDatabase.EXPECT().Get(gomock.Any(), member.TelegramIDInt()).Return(member, nil)
			sender = &tele.User{
				ID:        12345,
				FirstName: "Victor",
				LastName:  "Perez",
				Username:  "MetalBlueberry",
			}
			mockTeleContext.EXPECT().Sender().Return(sender).AnyTimes()

			mockTeleContext.EXPECT().Update().Return(tele.Update{}).AnyTimes()
			mockTeleContext.EXPECT().Get(gomock.Any()).Return(context.Background()).AnyTimes()

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
					Unixtime: time.Now().Unix(),
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
			It("Should reply with game details", func() {
				mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
					{
						ID:   "1",
						Name: "Game1",
					},
				}, nil)
				text := "Game1"
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Text:   text,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Unixtime: time.Now().Unix(),
				}).AnyTimes()
				mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					return nil
				})
				err := h.OnText(mockTeleContext)
				Expect(err).ToNot(HaveOccurred())
			})
			Describe("If the state is UpdateComment", func() {
				BeforeEach(func() {
					game := acnil.Game{
						ID:   "1",
						Name: "Game1",
					}
					member.State.SetUpdateComment(game)
					mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&game, nil)
				})
				It("Must update game comment", func() {
					text := "New Game comment"
					mockTeleContext.EXPECT().Message().Return(&tele.Message{
						Sender: sender,
						Text:   text,
						Chat: &tele.Chat{
							Type: tele.ChatPrivate,
						},
						Unixtime: time.Now().Unix(),
					}).AnyTimes()
					mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
					mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, game acnil.Game) error {
						Expect(game.Comments).To(Equal("New Game comment"))
						return nil
					})
					mockMembersDatabase.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, member acnil.Member) error {
						Expect(member.State.Action).To(BeEmpty())
						Expect(member.State.Data).To(BeEmpty())
						return nil
					})

					mockTeleContext.EXPECT().Send(ContainsString(text), gomock.Any()).Times(1)
					mockTeleContext.EXPECT().Respond().Times(1)
					err := h.OnText(mockTeleContext)
					Expect(err).ToNot(HaveOccurred())
				})

			})
		})
		Describe("When Text is sent with multiple lines", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
					{
						ID:   "1",
						Name: "Game1",
					},
					{
						ID:   "2",
						Name: "Game2",
					},
					{
						ID:   "3",
						Name: "Game3",
					},
				}, nil)
				text := "Game1\nGame2\n3"
				mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Text:   text,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Unixtime: time.Now().Unix(),
				}).AnyTimes()
			})
			It("Should reply with game details for each line", func() {
				mockTeleContext.EXPECT().Send(ContainsString("Game1"), gomock.Any()).Times(1).Do(func(sent string, opt ...interface{}) {
					Expect(sent).To(ContainSubstring("Game1"))
					Expect(sent).To(ContainSubstring("Game2"))
					Expect(sent).To(ContainSubstring("Game3"))
				})
				err := h.OnText(mockTeleContext)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Describe("When Text is an ID", func() {
			Describe("with leading 0", func() {
				BeforeEach(func() {
					text := "01"
					mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
						{
							ID:   "1",
							Name: "Game1",
						},
					}, nil)
					mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
					mockTeleContext.EXPECT().Message().Return(&tele.Message{
						Sender: sender,
						Text:   text,
						Chat: &tele.Chat{
							Type: tele.ChatPrivate,
						},
						Unixtime: time.Now().Unix(),
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
			Describe("without leading 0", func() {
				BeforeEach(func() {
					text := "1"
					mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
						{
							ID:   "1",
							Name: "Game1",
						},
					}, nil)
					mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
					mockTeleContext.EXPECT().Message().Return(&tele.Message{
						Sender: sender,
						Text:   text,
						Chat: &tele.Chat{
							Type: tele.ChatPrivate,
						},
						Unixtime: time.Now().Unix(),
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
		})
		Describe("When Text is an ID command", func() {
			Describe("with leading 0", func() {
				BeforeEach(func() {
					text := "/01"
					mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
						{
							ID:   "1",
							Name: "Game1",
						},
					}, nil)
					mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
					mockTeleContext.EXPECT().Message().Return(&tele.Message{
						Sender: sender,
						Text:   text,
						Chat: &tele.Chat{
							Type: tele.ChatPrivate,
						},
						Unixtime: time.Now().Unix(),
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
			Describe("without leading 0", func() {
				BeforeEach(func() {
					text := "/1"
					mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
						{
							ID:   "1",
							Name: "Game1",
						},
					}, nil)
					mockTeleContext.EXPECT().Text().Return(text).AnyTimes()
					mockTeleContext.EXPECT().Message().Return(&tele.Message{
						Sender: sender,
						Text:   text,
						Chat: &tele.Chat{
							Type: tele.ChatPrivate,
						},
						Unixtime: time.Now().Unix(),
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
		})
		Describe("When an user attempts to take a game that is available", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
					ID:   "1",
					Name: "Game1",
				}, nil)

				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:   "1",
						Name: "Game1",
					}.Card(),
				}).AnyTimes()
				mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{})).Do(func(ctx context.Context, g acnil.Game) {
					Expect(g.ID).To(Equal("1"))
					Expect(g.Name).To(Equal("Game1"))
					Expect(g.Holder).To(Equal(member.Nickname))
					Expect(g.ReturnDateFormula).To(HaveValue(Equal("=INDIRECT(ADDRESS(ROW();COLUMN()-1))+21")))

				})
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

				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:     "1",
						Name:   "Game1",
						Holder: "Other Person",
					}.Card(),
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
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(nil, nil)
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:     "1",
						Name:   "Game1",
						Holder: "Other Person",
					}.Card(),
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
					ID:       "1",
					Name:     "Game1",
					Holder:   member.Nickname,
					TakeDate: time.Date(2023, 2, 11, 0, 0, 0, 0, time.UTC),
				}, nil)

				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:   "1",
						Name: "Game1",
					}.Card(),
				}).AnyTimes()
			})
			It("the game must be updated with empty holder", func() {
				mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{
					ID:   "1",
					Name: "Game1",
				})).Do(func(_ context.Context, g acnil.Game) {
					Expect(g.Name).To(Equal("Game1"))
					Expect(g.Holder).To(BeEmpty())
					Expect(g.TakeDate).To(BeZero())
				})
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					Expect(sent).ToNot(ContainSubstring(member.Nickname))
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

				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:   "1",
						Name: "Game1",
					}.Card(),
				}).AnyTimes()

			})
			It("the game must be updated with empty holder", func() {
				mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{
					ID:   "1",
					Name: "Game1",
				})).Do(func(_ context.Context, g acnil.Game) {
					Expect(g.Name).To(Equal("Game1"))
					Expect(g.Holder).To(BeEmpty())
					Expect(g.TakeDate).To(BeZero())
				})
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					Expect(sent).ToNot(ContainSubstring(member.Nickname))
					return nil
				})
				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnReturn(mockTeleContext)
				Expect(err).To(BeNil())
			})

		})
		Describe("When an user list games held by him", func() {
			It("must list only games held", func() {
				mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
					{
						ID:     "1",
						Name:   "Game1",
						Holder: "Other User",
					},
					{
						ID:     "2",
						Name:   "Game2",
						Holder: member.Nickname,
					},
					{
						ID:     "3",
						Name:   "Game3",
						Holder: "Other User",
					},
				}, nil)
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game2"))
					Expect(sent).To(ContainSubstring(redDot))
					return nil
				})

				err := h.MyGames(mockTeleContext)
				Expect(err).To(BeNil())
			})
			It("must be case insensitive", func() {
				mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
					{
						ID:     "1",
						Name:   "Game1",
						Holder: "Other User",
					},
					{
						ID:     "2",
						Name:   "Game2",
						Holder: "metalblueberry",
					},
					{
						ID:     "3",
						Name:   "Game3",
						Holder: "Other User",
					},
				}, nil)
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game2"))
					Expect(sent).To(ContainSubstring(redDot))
					return nil
				})

				err := h.MyGames(mockTeleContext)
				Expect(err).To(BeNil())
			})
			It("must tolerate leading and trailing spaces on game holder field", func() {
				mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
					{
						ID:     "1",
						Name:   "Game1",
						Holder: "Other User",
					},
					{
						ID:     "2",
						Name:   "Game2",
						Holder: " metalblueberry ",
					},
					{
						ID:     "3",
						Name:   "Game3",
						Holder: "Other User",
					},
				}, nil)
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game2"))
					Expect(sent).To(ContainSubstring(redDot))
					return nil
				})

				err := h.MyGames(mockTeleContext)
				Expect(err).To(BeNil())
			})
			It("must tolerate leading special chars such as tildes", func() {
				mockGameDatabase.EXPECT().List(gomock.Any()).Return([]acnil.Game{
					{
						ID:     "1",
						Name:   "Game1",
						Holder: "Other User",
					},
					{
						ID:     "2",
						Name:   "Game2",
						Holder: "métalblueberry",
					},
					{
						ID:     "3",
						Name:   "Game3",
						Holder: "Other User",
					},
				}, nil)
				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game2"))
					Expect(sent).To(ContainSubstring(redDot))
					return nil
				})

				err := h.MyGames(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})

		Describe("When a page is requested for a game card", func() {
			BeforeEach(func() {
				mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
					ID:       "1",
					Name:     "Game1",
					Holder:   member.Nickname,
					TakeDate: time.Date(2023, 2, 11, 0, 0, 0, 0, time.UTC),
				}, nil)

				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:   "1",
						Name: "Game1",
					}.Card(),
				}).AnyTimes()
			})
			It("the page number 1 must be returned", func() {
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					Expect(sent).To(ContainSubstring(member.Nickname))
					Expect(opt[0]).To(BeAssignableToTypeOf(&tele.ReplyMarkup{}))

					buttons := ToOneDimension(opt[0].(*tele.ReplyMarkup).InlineKeyboard)
					Expect(buttons).To(ContainElement(WithButtonText(">")))
					return nil
				})
				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnGamePage(1)(mockTeleContext)
				Expect(err).To(BeNil())
			})
			It("the page number 2 must be returned", func() {
				mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					Expect(sent).To(ContainSubstring(member.Nickname))
					Expect(opt[0]).To(BeAssignableToTypeOf(&tele.ReplyMarkup{}))

					buttons := ToOneDimension(opt[0].(*tele.ReplyMarkup).InlineKeyboard)
					Expect(buttons).To(ContainElement(WithButtonText("<")))
					return nil
				})
				mockTeleContext.EXPECT().Respond(gomock.Any())

				err := h.OnGamePage(2)(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})

		Describe("When a game is switched locations", func() {

			BeforeEach(func() {

				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:   "1",
						Name: "Game1",
					}.Card(),
				}).AnyTimes()
			})
			Describe("When it was in Gamonal", func() {
				BeforeEach(func() {
					mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
						ID:       "1",
						Name:     "Game1",
						Holder:   member.Nickname,
						TakeDate: time.Date(2023, 2, 11, 0, 0, 0, 0, time.UTC),
						Location: string(acnil.LocationGamonal),
					}, nil)
				})
				It("Must be moved to Centro", func() {
					mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{
						ID:   "1",
						Name: "Game1",
					})).Do(func(_ context.Context, g acnil.Game) {
						Expect(g.Name).To(Equal("Game1"))
						Expect(g.Location).To(Equal(string(acnil.LocationCentro)))
					})

					mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
						Expect(sent).To(ContainSubstring("Game1"))
						Expect(sent).To(ContainSubstring(member.Nickname))
						Expect(sent).To(ContainSubstring(string(acnil.LocationCentro)))

						Expect(opt[0]).To(BeAssignableToTypeOf(&tele.ReplyMarkup{}))

						buttons := ToOneDimension(opt[0].(*tele.ReplyMarkup).InlineKeyboard)
						Expect(buttons).To(ContainElement(WithButtonText("Mover a Gamonal")))
						return nil
					})
					mockTeleContext.EXPECT().Respond(gomock.Any())

					err := h.OnSwitchLocation(mockTeleContext)
					Expect(err).To(BeNil())
				})
			})
			Describe("When it was in Centro", func() {
				BeforeEach(func() {
					mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
						ID:       "1",
						Name:     "Game1",
						Holder:   member.Nickname,
						TakeDate: time.Date(2023, 2, 11, 0, 0, 0, 0, time.UTC),
						Location: string(acnil.LocationCentro),
					}, nil)
				})
				It("Must be moved to Gamonal", func() {
					mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{
						ID:   "1",
						Name: "Game1",
					})).Do(func(_ context.Context, g acnil.Game) {
						Expect(g.Name).To(Equal("Game1"))
						Expect(g.Location).To(Equal(string(acnil.LocationGamonal)))
					})

					mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
						Expect(sent).To(ContainSubstring("Game1"))
						Expect(sent).To(ContainSubstring(member.Nickname))
						Expect(sent).To(ContainSubstring(string(acnil.LocationGamonal)))

						Expect(opt[0]).To(BeAssignableToTypeOf(&tele.ReplyMarkup{}))

						buttons := ToOneDimension(opt[0].(*tele.ReplyMarkup).InlineKeyboard)
						Expect(buttons).To(ContainElement(WithButtonText("Mover al Centro")))
						return nil
					})
					mockTeleContext.EXPECT().Respond(gomock.Any())

					err := h.OnSwitchLocation(mockTeleContext)
					Expect(err).To(BeNil())
				})
			})
			Describe("When it was in an unknown location", func() {
				BeforeEach(func() {
					mockGameDatabase.EXPECT().Get(gomock.Any(), "1", "Game1").Return(&acnil.Game{
						ID:       "1",
						Name:     "Game1",
						Holder:   member.Nickname,
						TakeDate: time.Date(2023, 2, 11, 0, 0, 0, 0, time.UTC),
						Location: "Invalid location",
					}, nil)
				})
				It("Must be moved to Gamonal", func() {
					mockGameDatabase.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(acnil.Game{
						ID:   "1",
						Name: "Game1",
					})).Do(func(_ context.Context, g acnil.Game) {
						Expect(g.Name).To(Equal("Game1"))
						Expect(g.Location).To(Equal(string(acnil.LocationGamonal)))
					})

					mockTeleContext.EXPECT().Edit(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
						Expect(sent).To(ContainSubstring("Game1"))
						Expect(sent).To(ContainSubstring(member.Nickname))
						Expect(sent).To(ContainSubstring(string(acnil.LocationGamonal)))

						Expect(opt[0]).To(BeAssignableToTypeOf(&tele.ReplyMarkup{}))

						buttons := ToOneDimension(opt[0].(*tele.ReplyMarkup).InlineKeyboard)
						Expect(buttons).To(ContainElement(WithButtonText("Mover al Centro")))
						return nil
					})
					mockTeleContext.EXPECT().Respond(gomock.Any())

					err := h.OnSwitchLocation(mockTeleContext)
					Expect(err).To(BeNil())
				})
			})
		})
		Describe("When the button to update comments is used", func() {
			BeforeEach(func() {
				mockTeleContext.EXPECT().Message().Return(&tele.Message{
					Sender: sender,
					Chat: &tele.Chat{
						Type: tele.ChatPrivate,
					},
					Text: acnil.Game{
						ID:   "1",
						Name: "Game1",
					}.Card(),
				}).AnyTimes()
			})
			It("Must ask for the new comment", func() {
				mockMembersDatabase.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, member acnil.Member) error {
					Expect(member.State.Data).To(ContainSubstring("Game1"))
					return nil
				})
				mockTeleContext.EXPECT().Respond(gomock.Any())

				mockTeleContext.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(sent string, opt ...interface{}) error {
					Expect(sent).To(ContainSubstring("Game1"))
					return nil
				})
				err := h.OnUpdateCommentButton(mockTeleContext)
				Expect(err).To(BeNil())
			})
		})
	})

})
