package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type reportDocument struct {
	ID          string    `bson:"_id"`
	ProcessID   string    `bson:"process_id"`
	Components  []string  `bson:"components"`
	Risks       []string  `bson:"risks"`
	Recs        []string  `bson:"recommendations"`
	RawResponse string    `bson:"raw_response"`
	CreatedAt   time.Time `bson:"created_at"`
}

type ReportRepository struct {
	coll *mongo.Collection
}

func NewReportRepository(db *mongo.Database) *ReportRepository {
	return &ReportRepository{coll: db.Collection("reports")}
}

// EnsureIndexes cria índice no process_id para buscas futuras.
func (r *ReportRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "process_id", Value: 1}},
		Options: options.Index().SetBackground(true),
	})
	return err
}

func (r *ReportRepository) Save(ctx context.Context, report *domain.Report) error {
	doc := reportDocument{
		ID:          report.ID,
		ProcessID:   report.ProcessID,
		Components:  report.Analysis.Components,
		Risks:       report.Analysis.Risks,
		Recs:        report.Analysis.Recommendations,
		RawResponse: report.RawResponse,
		CreatedAt:   report.CreatedAt,
	}
	_, err := r.coll.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("mongo insert: %w", err)
	}
	return nil
}

func (r *ReportRepository) FindByID(ctx context.Context, id string) (*domain.Report, error) {
	var doc reportDocument
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrReportNotFound
		}
		return nil, fmt.Errorf("mongo find: %w", err)
	}
	return docToDomain(&doc), nil
}

func docToDomain(doc *reportDocument) *domain.Report {
	return &domain.Report{
		ID:        doc.ID,
		ProcessID: doc.ProcessID,
		Analysis: domain.Analysis{
			Components:      doc.Components,
			Risks:           doc.Risks,
			Recommendations: doc.Recs,
		},
		RawResponse: doc.RawResponse,
		CreatedAt:   doc.CreatedAt,
	}
}
