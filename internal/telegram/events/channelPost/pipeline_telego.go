package channelpost

import (
	"context"
	"fmt"

	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/telegram/executor"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

// ProcessingContextTelego holds the entire state of a message as it moves through the pipeline using telego.
type ProcessingContextTelego struct {
	// Original Request Data
	Ctx    context.Context
	Bot    *telego.Bot
	Update telego.Update

	// Derived Data
	MessageType MessageType
	Channel     *dbmodels.Channel
	Permissions *PermissionCheckResult

	// Transformation State
	OriginalCaption    string
	FormattedText      string
	DisableLinkPreview bool
	FinalButtons       []dbmodels.Button
	FinalKeyboard      *telego.InlineKeyboardMarkup

	// Entity-based Caption (MTProto)
	// Quando preenchido, o dispatcher deve usar entities ao inves de parseMode HTML.
	FinalEntities    string // JSON combinado dos MessageEntityDTO (post + caption)
	PostEntitiesJSON string // JSON dos MessageEntityDTO do post original (para combinacao)

	// Media Group State (for albums)
	IsMediaGroup  bool
	MediaGroupID  string
	GroupMessages []MediaMessageTelego

	// Execution Control
	Pipeline     *PipelineTelego
	StopPipeline bool // If true, remaining stages are skipped
	Error        error

	// Executor
	ExecutorFactory *executor.ExecutorFactory
}

type MediaMessageTelego struct {
	MessageID       int
	FileID          string
	HasCaption      bool
	Caption         string
	CaptionEntities []telego.MessageEntity
}

// StageTelego defines a single step in the processing pipeline using telego.
type StageTelego func(ctx *ProcessingContextTelego) error

// PipelineTelego orchestrates the execution of multiple stages for telego.
type PipelineTelego struct {
	Name   string
	stages []StageTelego
}

// NewPipelineTelego initializes a pipeline with the provided stages.
func NewPipelineTelego(name string, stages ...StageTelego) *PipelineTelego {
	return &PipelineTelego{
		Name:   name,
		stages: stages,
	}
}

// Execute runs the pipeline for a given context starting from the first stage.
func (p *PipelineTelego) Execute(ctx *ProcessingContextTelego) error {
	return p.ExecuteFrom(ctx, 0)
}

// ExecuteFrom runs the pipeline starting from a specific stage index.
func (p *PipelineTelego) ExecuteFrom(ctx *ProcessingContextTelego, startIndex int) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("PIPELINE", "[%s] Recovered from panic: %v", p.Name, r)
			ctx.Error = fmt.Errorf("pipeline panic: %v", r)
		}
	}()

	for i := startIndex; i < len(p.stages); i++ {
		if ctx.StopPipeline {
			logger.Bot("⏹️ Pipeline [%s] stopped at stage %d", p.Name, i+1)
			break
		}

		if err := p.stages[i](ctx); err != nil {
			ctx.Error = err
			logger.Error("PIPELINE", "[%s] Error at stage %d: %v", p.Name, i+1, err)
			return err
		}
	}

	return nil
}

// NewProcessingContextTelego creates a fresh context for a telego update.
func NewProcessingContextTelego(ctx context.Context, b *telego.Bot, update telego.Update, p *PipelineTelego) *ProcessingContextTelego {
	return &ProcessingContextTelego{
		Ctx:      ctx,
		Bot:      b,
		Update:   update,
		Pipeline: p,
	}
}
