package acnil_test

import (
	"time"

	"github.com/acnil/acnil-bot/pkg/acnil"
	. "github.com/acnil/acnil-bot/pkg/acnil/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/telebot.v3"
)

var _ = Describe("A game card", func() {
	var (
		game acnil.Game
	)
	BeforeEach(func() {
		game = acnil.Game{
			ID:   "1",
			Name: "Brass Brimmingan",
		}
	})

	Describe("For an authorised member", func() {
		var (
			member acnil.Member
		)
		BeforeEach(func() {
			member = acnil.Member{
				Nickname:    "Metalblueberry",
				TelegramID:  "1234",
				Permissions: acnil.PermissionYes,
			}
		})

		Describe("For an available game", func() {
			BeforeEach(func() {
				game.Holder = ""
			})

			It("Must contain the game name", func() {
				card := game.Card()
				Expect(card).To(ContainSubstring(game.Name))
			})

			It("Must NOT contain return button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Devolver")))
			})
			It("Must contain take button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).To(ContainElement(WithButtonText("Tomar Prestado")))
			})
			It("Must not have a button to increase the time by a few days", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Dar mas tiempo")))
			})
			It("Must contain > button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).To(ContainElement(WithButtonText(">")))
			})
			Describe("Ïf return date is set but holder is not", func() {
				BeforeEach(func() {
					game.ReturnDate = time.Now().Add(-24 * 30 * time.Hour)
				})
				It("Must not have a button to increase the time by a few days", func() {
					buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
					Expect(buttons).ToNot(ContainElement(WithButtonText("Dar mas tiempo")))
				})
				It("Must append data to all buttons", func() {
					buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
					for _, button := range buttons {
						Expect(button).To(WithButtonData(button.Data))
					}
				})
			})
			Describe("in the 2nd button page", func() {
				Describe("When it is in Gamonal", func() {
					BeforeEach(func() {
						game.Location = string(acnil.LocationGamonal)
					})
					It("Must contain Mover al Centro button", func() {
						buttons := ToOneDimension(game.ButtonsForPage(member, 2).InlineKeyboard)
						var button telebot.InlineButton
						Expect(buttons).To(ContainElement(WithButtonText("Mover al Centro"), &button))
					})
				})
				Describe("When it is in Centro", func() {
					BeforeEach(func() {
						game.Location = string(acnil.LocationCentro)
					})
					It("Must contain Mover a Gamonal button", func() {
						buttons := ToOneDimension(game.ButtonsForPage(member, 2).InlineKeyboard)
						var button telebot.InlineButton
						Expect(buttons).To(ContainElement(WithButtonText("Mover a Gamonal"), &button))
					})
				})
				It("Must contain Actualizar comentario button", func() {
					buttons := ToOneDimension(game.ButtonsForPage(member, 2).InlineKeyboard)
					Expect(buttons).To(ContainElement(WithButtonText("Actualizar comentario")))
				})
				It("Must contain < button", func() {
					buttons := ToOneDimension(game.ButtonsForPage(member, 2).InlineKeyboard)
					Expect(buttons).To(ContainElement(WithButtonText("<")))
				})
				It("Must append data to all buttons", func() {
					buttons := ToOneDimension(game.ButtonsForPage(member, 2).InlineKeyboard)
					for _, button := range buttons {
						Expect(button).To(WithButtonData(button.Data))
					}
				})
			})
		})

		Describe("for a game held by himself", func() {
			BeforeEach(func() {
				game.Holder = member.Nickname
			})

			It("Must contain the game name", func() {
				card := game.Card()
				Expect(card).To(ContainSubstring(game.Name))
			})
			It("Must contain the holder name", func() {
				card := game.Card()
				Expect(card).To(ContainSubstring(member.Nickname))
			})
			It("Must contain return button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).To(ContainElement(WithButtonText("Devolver")))
			})
			It("Must not contain take button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Tomar Prestado")))
			})

			Describe("For a game with return date", func() {
				Describe("that has expired 48h ago", func() {
					BeforeEach(func() {
						game.ReturnDate = time.Now().Add(-time.Hour * 48)
					})
					It("should have a warning icon", func() {
						card := game.Card()
						Expect(card).To(ContainSubstring("⚠️"))
					})
					It("should have a button to increase the time by a few days", func() {
						buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
						Expect(buttons).To(ContainElement(WithButtonText("Dar mas tiempo")))
					})
				})
			})
		})
		Describe("for a game held by other person", func() {
			BeforeEach(func() {
				game.Holder = "Other User"
			})

			It("Must contain the game name", func() {
				card := game.Card()
				Expect(card).To(ContainSubstring(game.Name))
			})
			It("Must contain the other holder name", func() {
				card := game.Card()
				Expect(card).To(ContainSubstring(game.Holder))
			})
			It("Must contain return button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).To(ContainElement(WithButtonText("Devolver")))
			})
			It("Must NOT contain take button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Tomar Prestado")))
			})

			It("should NOT have a button to increase the time by a few days", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Dar mas tiempo")))
			})

			Describe("For a game with return date", func() {
				Describe("that has expired 48h ago", func() {
					BeforeEach(func() {
						game.ReturnDate = time.Now().Add(-time.Hour * 48)
					})
					It("should have a warning icon", func() {
						card := game.Card()
						Expect(card).To(ContainSubstring("⚠️"))
					})
					It("should NOT have a button to increase the time by a few days", func() {
						buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
						Expect(buttons).ToNot(ContainElement(WithButtonText("Dar mas tiempo")))
					})

				})
			})
		})

		Describe("with BGG Data", func() {
			BeforeEach(func() {
				game.BGG = "123"
			})
			It("Must contain more info button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).To(ContainElement(WithButtonText("Mas información")))
			})
		})
		Describe("without BGG Data", func() {
			BeforeEach(func() {
				game.BGG = ""
			})
			It("Must Not contain more info button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Mas información")))
			})
		})
	})

	Describe("For an admin member", func() {
		var (
			member acnil.Member
		)
		BeforeEach(func() {
			member = acnil.Member{
				Nickname:    "Metalblueberry",
				TelegramID:  "1234",
				Permissions: acnil.PermissionAdmin,
			}
		})

		Describe("for an available game", func() {
			BeforeEach(func() {
				game.Holder = ""
			})
			Describe("that has the return date set incorrectly", func() {

				BeforeEach(func() {
					game.ReturnDate = time.Now().Add(-30 * 24 * time.Hour)
				})
				It("must not have a button to increase the time by a few days", func() {
					buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
					Expect(buttons).ToNot(ContainElement(WithButtonText("Dar mas tiempo")))
				})
			})

		})

		Describe("for a game held by other person", func() {
			BeforeEach(func() {
				game.Holder = "Other User"
			})

			It("Must contain the game name", func() {
				card := game.Card()
				Expect(card).To(ContainSubstring(game.Name))
			})
			It("Must contain the other holder name", func() {
				card := game.Card()
				Expect(card).To(ContainSubstring(game.Holder))
			})
			It("Must contain return button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).To(ContainElement(WithButtonText("Devolver")))
			})
			It("Must NOT contain take button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Tomar Prestado")))
			})

			It("should NOT have a button to increase the time by a few days", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Dar mas tiempo")))
			})

			Describe("For a game with return date", func() {
				Describe("that has expired 48h ago", func() {
					BeforeEach(func() {
						game.ReturnDate = time.Now().Add(-time.Hour * 48)
					})
					It("should have a warning icon", func() {
						card := game.Card()
						Expect(card).To(ContainSubstring("⚠️"))
					})
					It("should have a button to increase the time by a few days", func() {
						buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
						Expect(buttons).To(ContainElement(WithButtonText("Dar mas tiempo")))
					})

				})
			})

		})
	})
})

var _ = Describe("A game line", func() {
	var (
		gameLine string
	)
	Describe("without user", func() {
		BeforeEach(func() {
			gameLine = "🟢 0505: Brass Birmingham"
		})

		It("must parse the ID", func() {
			game, err := acnil.NewGameFromLine(gameLine)
			Expect(err).To(BeNil())
			Expect(game.ID).To(Equal("505"))
		})
		It("must parse the name", func() {
			game, err := acnil.NewGameFromLine(gameLine)
			Expect(err).To(BeNil())
			Expect(game.Name).To(Equal("Brass Birmingham"))
		})
		It("must parse the Holder as empty", func() {
			game, err := acnil.NewGameFromLine(gameLine)
			Expect(err).To(BeNil())
			Expect(game.Holder).To(BeEmpty())
		})

	})
	Describe("with user", func() {
		BeforeEach(func() {
			gameLine = "🔴 0505: Brass Birmingham (Metalblueberry)"
		})
		It("must parse the ID", func() {
			game, err := acnil.NewGameFromLine(gameLine)
			Expect(err).To(BeNil())
			Expect(game.ID).To(Equal("505"))
		})
		It("must parse the name", func() {
			game, err := acnil.NewGameFromLine(gameLine)
			Expect(err).To(BeNil())
			Expect(game.Name).To(Equal("Brass Birmingham"))
		})
		It("must parse the Holder", func() {
			game, err := acnil.NewGameFromLine(gameLine)
			Expect(err).To(BeNil())
			Expect(game.Holder).To(Equal("Metalblueberry"))
		})

	})

})

var _ = Describe("A game card", func() {
	var (
		game = acnil.Game{
			ID:       "123",
			Name:     "TestGame",
			Location: string(acnil.LocationCentro),
			Comments: "This is a test game",
			Holder:   "",
		}
	)
	It("Must be parsed from its card", func() {
		gameCard := game.Card()
		g, err := acnil.NewGameFromCard(gameCard)
		Expect(err).To(BeNil())
		Expect(g.ID).To(Equal(game.ID))
		Expect(g.Name).To(Equal(game.Name))
	})

	It("Must be parsed from its morecard", func() {
		gameCard := game.MoreCard()
		g, err := acnil.NewGameFromCard(gameCard)
		Expect(err).To(BeNil())
		Expect(g.ID).To(Equal(game.ID))
		Expect(g.Name).To(Equal(game.Name))
	})

})
