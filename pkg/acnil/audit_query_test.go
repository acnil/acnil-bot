package acnil_test

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/acnil/mock_acnil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=audit_query.go -destination mock_acnil/audit_query.go
var _ = Describe("AuditQuery: ", func() {

	var (
		ctrl              *gomock.Controller
		mockAuditDatabase *mock_acnil.MockAuditDatabase
		audit             *acnil.AuditQuery
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockAuditDatabase = mock_acnil.NewMockAuditDatabase(ctrl)

		audit = &acnil.AuditQuery{
			AuditDB: mockAuditDatabase,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("When history for a game is requested", func() {
		var (
			auditedEntries = []acnil.AuditEntry{}
		)
		BeforeEach(func() {
			auditedEntries = []acnil.AuditEntry{
				{
					Timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					ID:        "1",
					Name:      "Game1",
					Location:  "Centro",
					Holder:    "",
				},
				{
					Timestamp: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
					ID:        "1",
					Name:      "Game1",
					Location:  "Centro",
					Holder:    "MetalBlueberry",
				},
				{
					Timestamp: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
					ID:        "1",
					Name:      "Game1",
					Location:  "Centro",
					Holder:    "OtherMember",
				},
				{
					Timestamp: time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC),
					ID:        "2",
					Name:      "Game2",
					Location:  "Gamonal",
					Holder:    "",
				},
				{
					Timestamp: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC),
					ID:        "1",
					Name:      "Game1",
					Location:  "Gamonal",
					Holder:    "",
				},
				{
					Timestamp: time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC),
					ID:        "2",
					Name:      "Game2",
					Location:  "Gamonal",
					Holder:    "MetalBlueberry",
				},
			}

			mockAuditDatabase.EXPECT().List(gomock.Any()).Return(auditedEntries, nil)
		})
		It("Must list all items for a game", func() {
			game := acnil.Game{
				ID:   "1",
				Name: "Game1",
			}
			list, err := audit.Find(context.Background(), acnil.Query{
				Game: &game,
			})
			Expect(err).To(BeNil())

			Expect(list).To(HaveLen(4))
			Expect(list).To(Equal([]acnil.AuditEntry{
				auditedEntries[0],
				auditedEntries[1],
				auditedEntries[2],
				auditedEntries[4],
			}))
		})
		It("Must limit game entries if requested, starting by older entries", func() {
			game := acnil.Game{
				ID:   "1",
				Name: "Game1",
			}
			list, err := audit.Find(context.Background(), acnil.Query{
				Game:  &game,
				Limit: 3,
			})
			Expect(err).To(BeNil())

			Expect(list).To(HaveLen(3))
			Expect(list).To(ContainElements(
				auditedEntries[1],
				auditedEntries[2],
				auditedEntries[4],
			))
		})
		It("Must display data only within a time range but still respect the limit", func() {
			game := acnil.Game{
				ID:   "1",
				Name: "Game1",
			}
			list, err := audit.Find(context.Background(), acnil.Query{
				Game:  &game,
				Limit: 2,
				From:  time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				To:    time.Date(2023, 2, 2, 0, 0, 0, 0, time.UTC),
			})
			Expect(err).To(BeNil())

			Expect(list).To(HaveLen(2))
			Expect(list).To(ContainElements(
				auditedEntries[2],
				auditedEntries[4],
			))
		})
		It("Must display data for a member", func() {
			list, err := audit.Find(context.Background(), acnil.Query{
				Member: &acnil.Member{
					Nickname: "Metalblueberry",
				},
			})
			Expect(err).To(BeNil())

			Expect(list).To(HaveLen(2))
			Expect(list).To(ContainElements(
				auditedEntries[1],
				auditedEntries[5],
			))
		})

	})
})
