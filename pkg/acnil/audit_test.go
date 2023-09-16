package acnil_test

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/acnil/mock_acnil"
)

//go:generate mockgen -source=audit.go -destination mock_acnil/audit.go
var _ = Describe("Audit: ", func() {

	var (
		ctrl              *gomock.Controller
		mockGameDatabase  *mock_acnil.MockROGameDatabase
		mockAuditDatabase *mock_acnil.MockAuditDatabase
		audit             *acnil.Audit
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockGameDatabase = mock_acnil.NewMockROGameDatabase(ctrl)
		mockAuditDatabase = mock_acnil.NewMockAuditDatabase(ctrl)

		audit = &acnil.Audit{
			AuditDB: mockAuditDatabase,
			GameDB:  mockGameDatabase,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("When Audit is initialised", func() {
		var (
			database = []acnil.Game{}
		)

		BeforeEach(func() {
			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
			mockGameDatabase.EXPECT().List(gomock.Any()).Return(database, nil)
			mockAuditDatabase.EXPECT().List(gomock.Any()).Return([]acnil.AuditEntry{}, nil)
		})
		It("Should generate an event per game", func() {
			auditedEntries := []acnil.AuditEntry{}
			mockAuditDatabase.EXPECT().Append(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, entries []acnil.AuditEntry) {
				auditedEntries = entries
			}).Return(nil).AnyTimes()

			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			Expect(auditedEntries).To(HaveLen(3))

			for _, gameInDatabase := range database {
				Expect(auditedEntries).To(ContainElement(acnil.NewAuditEntry(gameInDatabase, acnil.AuditEntryTypeNew)))
			}
		})

	})

	Describe("When A new game is added", func() {
		var (
			database       = []acnil.Game{}
			auditedEntries = []acnil.AuditEntry{}
		)

		BeforeEach(func() {
			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
			mockGameDatabase.EXPECT().List(gomock.Any()).DoAndReturn(
				func(_ context.Context) ([]acnil.Game, error) {
					return database, nil
				}).AnyTimes()
			mockAuditDatabase.EXPECT().List(gomock.Any()).Return([]acnil.AuditEntry{}, nil)

			mockAuditDatabase.EXPECT().Append(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, entries []acnil.AuditEntry) {
				auditedEntries = entries
			}).Return(nil).AnyTimes()

			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "4",
					Name:       "NewGame",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
		})
		It("should generate a new game audit event", func() {
			// reset audited entries
			auditedEntries = []acnil.AuditEntry{}

			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			Expect(auditedEntries).To(HaveLen(1))

			Expect(auditedEntries[0]).To(Equal(acnil.NewAuditEntry(database[2], acnil.AuditEntryTypeNew)))
		})
	})

	Describe("When A game is removed", func() {
		var (
			database       = []acnil.Game{}
			auditedEntries = []acnil.AuditEntry{}
			removedGame    = acnil.Game{}
		)

		BeforeEach(func() {
			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
			mockGameDatabase.EXPECT().List(gomock.Any()).DoAndReturn(
				func(_ context.Context) ([]acnil.Game, error) {
					return database, nil
				}).AnyTimes()
			mockAuditDatabase.EXPECT().List(gomock.Any()).Return([]acnil.AuditEntry{}, nil)

			mockAuditDatabase.EXPECT().Append(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, entries []acnil.AuditEntry) {
				auditedEntries = entries
			}).Return(nil).AnyTimes()

			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			removedGame = acnil.Game{
				Row:        "",
				ID:         "2",
				Name:       "Game2",
				Location:   "Gamonal",
				Holder:     "",
				Comments:   "",
				TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
				ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
				Price:      "13.00",
				Publisher:  "OpenSource",
				BGG:        "",
			}
			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
		})
		It("should generate a removed game audit event", func() {
			// reset audited entries
			auditedEntries = []acnil.AuditEntry{}

			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			Expect(auditedEntries).To(HaveLen(1))

			Expect(auditedEntries[0]).To(Equal(acnil.NewAuditEntry(removedGame, acnil.AuditEntryTypeRemoved)))
		})
	})

	Describe("When A game is modified", func() {
		var (
			database       = []acnil.Game{}
			auditedEntries = []acnil.AuditEntry{}
		)

		BeforeEach(func() {
			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
			mockGameDatabase.EXPECT().List(gomock.Any()).DoAndReturn(
				func(_ context.Context) ([]acnil.Game, error) {
					return database, nil
				}).AnyTimes()
			mockAuditDatabase.EXPECT().List(gomock.Any()).Return([]acnil.AuditEntry{}, nil)

			mockAuditDatabase.EXPECT().Append(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, entries []acnil.AuditEntry) {
				auditedEntries = entries
			}).Return(nil).AnyTimes()

			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
		})
		It("should generate a game updated audit events", func() {
			// reset audited entries
			auditedEntries = []acnil.AuditEntry{}

			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			Expect(auditedEntries).To(HaveLen(2))

			Expect(auditedEntries).To(ContainElement(
				acnil.NewAuditEntry(database[1], acnil.AuditEntryTypeUpdate),
			))
			Expect(auditedEntries).To(ContainElement(
				acnil.NewAuditEntry(database[2], acnil.AuditEntryTypeUpdate),
			))
		})
	})

	Describe("When Audit already contains entries", func() {
		var (
			database       = []acnil.Game{}
			auditedEntries = []acnil.AuditEntry{}
		)

		BeforeEach(func() {
			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}

			preAuditedEntries := []acnil.AuditEntry{}
			for _, game := range database {
				preAuditedEntries = append(preAuditedEntries, acnil.NewAuditEntry(game, acnil.AuditEntryTypeNew))
			}

			mockGameDatabase.EXPECT().List(gomock.Any()).DoAndReturn(
				func(_ context.Context) ([]acnil.Game, error) {
					return database, nil
				}).AnyTimes()
			mockAuditDatabase.EXPECT().List(gomock.Any()).Return(preAuditedEntries, nil)

			mockAuditDatabase.EXPECT().Append(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, entries []acnil.AuditEntry) {
				auditedEntries = entries
			}).Return(nil).AnyTimes()

			database = []acnil.Game{
				{
					Row:        "",
					ID:         "1",
					Name:       "Game1",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "2",
					Name:       "Game2",
					Location:   "Gamonal",
					Holder:     "Victor",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
				{
					Row:        "",
					ID:         "3",
					Name:       "Game3",
					Location:   "Centro",
					Holder:     "",
					Comments:   "",
					TakeDate:   time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					ReturnDate: time.Date(2023, 01, 13, 0, 0, 0, 0, time.UTC),
					Price:      "13.00",
					Publisher:  "OpenSource",
					BGG:        "",
				},
			}
		})

		It("should generate a game updated audit events", func() {
			// reset audited entries
			err := audit.Do(context.Background())
			Expect(err).To(BeNil())

			Expect(auditedEntries).To(HaveLen(2))

			Expect(auditedEntries).To(ContainElement(
				acnil.NewAuditEntry(database[1], acnil.AuditEntryTypeUpdate),
			))
			Expect(auditedEntries).To(ContainElement(
				acnil.NewAuditEntry(database[2], acnil.AuditEntryTypeUpdate),
			))

			auditedEntries = []acnil.AuditEntry{}
			err = audit.Do(context.Background())
			Expect(err).To(BeNil())

			Expect(auditedEntries).To(HaveLen(0))
		})
	})

})
