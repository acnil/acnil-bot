package acnil_test

import (
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	. "github.com/metalblueberry/acnil-bot/pkg/acnil/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			It("Must NOT contain return button", func() {
				buttons := ToOneDimension(game.Buttons(member).InlineKeyboard)
				Expect(buttons).ToNot(ContainElement(WithButtonText("Devolver")))
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
