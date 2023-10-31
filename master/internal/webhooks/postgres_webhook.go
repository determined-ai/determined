package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	conf "github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/google/uuid"
)

type regexTriggers struct {
	re                 *regexp.Regexp
	triggerIDToTrigger map[TriggerID]*Trigger
}

// WebhookManager manages webhooks.
type WebhookManager struct {
	mu              sync.RWMutex
	regexToTriggers map[string]regexTriggers
}

// New creates a new webhook manager.
func New(ctx context.Context) (*WebhookManager, error) {
	var triggers []*Trigger
	if err := db.Bun().NewSelect().Model(&triggers).Relation("Webhook").
		Where("trigger_type = ?", TriggerTypeTaskLog).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("querying task logs triggers: %w", err)
	}

	m := &WebhookManager{
		regexToTriggers: make(map[string]regexTriggers),
	}
	if err := m.addTriggers(triggers); err != nil {
		return nil, fmt.Errorf("adding each trigger: %w", err)
	}

	return m, nil
}

func (l *WebhookManager) addTriggers(triggers []*Trigger) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, t := range triggers {
		if t.TriggerType != TriggerTypeTaskLog {
			continue
		}

		regex, ok := t.Condition[regexConditionKey].(string)
		if !ok {
			return fmt.Errorf(
				"expected webhook trigger to have regex in condition instead got %v", t.Condition)
		}

		if _, ok := l.regexToTriggers[regex]; !ok {
			compiled, err := regexp.Compile(regex)
			if err != nil {
				return fmt.Errorf("compiling regex %s: %w", regex, err)
			}

			l.regexToTriggers[regex] = regexTriggers{
				re:                 compiled,
				triggerIDToTrigger: make(map[TriggerID]*Trigger),
			}
		}

		l.regexToTriggers[regex].triggerIDToTrigger[t.ID] = t
	}

	return nil
}

func (l *WebhookManager) removeTriggers(triggers []*Trigger) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, t := range triggers {
		if t.TriggerType != TriggerTypeTaskLog {
			continue
		}

		regex, ok := t.Condition[regexConditionKey].(string)
		if !ok {
			log.Errorf(
				"expected webhook trigger to have regex in condition instead got %v deleting anyway",
				t.Condition)
			return nil
		}

		delete(l.regexToTriggers[regex].triggerIDToTrigger, t.ID)
	}
	return nil
}

func (l *WebhookManager) scanLogs(ctx context.Context, logs []*model.TaskLog) error {
	if len(logs) == 0 {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, log := range logs {
		if log.AgentID == nil {
			return fmt.Errorf("AgentID must be non nil to trigger webhooks in logs")
		}

		for _, cacheItem := range l.regexToTriggers {
			if cacheItem.re.MatchString(log.Log) {
				for _, t := range cacheItem.triggerIDToTrigger {
					if err := addTaskLogEvent(ctx,
						model.TaskID(log.TaskID), *log.AgentID, log.Log, t); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (l *WebhookManager) addWebhook(ctx context.Context, w *Webhook) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(w).Exec(ctx)
		if err != nil {
			return err
		}
		for _, t := range w.Triggers {
			t.WebhookID = w.ID
		}

		if len(w.Triggers) != 0 {
			_, err = tx.NewInsert().Model(&w.Triggers).Exec(ctx)
			if err != nil {
				return err
			}

			for _, t := range w.Triggers {
				t.Webhook = w
			}
			if err := l.addTriggers(w.Triggers); err != nil {
				return err
			}
		}
		return nil
	})
}

func (l *WebhookManager) deleteWebhook(ctx context.Context, id WebhookID) error {
	var ts []*Trigger
	if err := db.Bun().NewSelect().Model(&ts).Relation("Webhook").
		Where("webhook_id = ?", id).
		Scan(ctx, &ts); err != nil {
		return fmt.Errorf("getting webhook triggers to delete: %w", err)
	}

	if err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewDelete().Model((*Webhook)(nil)).Where("id = ?", id).Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting webhook id %d: %w", id, err)
		}

		if err := l.removeTriggers(ts); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("deleting webhooks: %w", err)
	}

	return nil
}

// GetWebhook returns a single Webhooks from the DB.
func GetWebhook(ctx context.Context, webhookID int) (*Webhook, error) {
	webhook := Webhook{}
	err := db.Bun().NewSelect().
		Model(&webhook).
		Relation("Triggers").
		Where("id = ?", webhookID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

// GetWebhooks returns all Webhooks from the DB.
func GetWebhooks(ctx context.Context) (Webhooks, error) {
	webhooks := Webhooks{}
	err := db.Bun().NewSelect().
		Model(&webhooks).
		Relation("Triggers").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

// ReportExperimentStateChanged adds webhook events to the queue.
// TODO(DET-8577): Remove unnecessary active config usage (remove the activeConfig parameter).
func ReportExperimentStateChanged(
	ctx context.Context, e model.Experiment, activeConfig expconf.ExperimentConfig,
) error {
	defer func() {
		if rec := recover(); rec != nil {
			log.Errorf("uncaught error in webhook report: %v", rec)
		}
	}()

	var ts []Trigger
	switch err := db.Bun().NewSelect().Model(&ts).Relation("Webhook").
		Where("trigger_type = ?", TriggerTypeStateChange).
		Where("condition->>'state' = ?", e.State).
		Scan(ctx); {
	case err != nil:
		return err
	case len(ts) == 0:
		return nil
	}

	var es []Event
	for _, t := range ts {
		p, err := generateEventPayload(
			ctx, t.Webhook.WebhookType, e, activeConfig, e.State, TriggerTypeStateChange,
		)
		if err != nil {
			return fmt.Errorf("error generating event payload: %w", err)
		}
		es = append(es, Event{Payload: p, URL: t.Webhook.URL})
	}
	if _, err := db.Bun().NewInsert().Model(&es).Exec(ctx); err != nil {
		return fmt.Errorf("report experiment state changed inserting event trigger: %w", err)
	}

	singletonShipper.Wake()
	return nil
}

func addTaskLogEvent(ctx context.Context,
	taskID model.TaskID, nodeName, triggeringLog string, trigger *Trigger,
) error {
	defer func() {
		if rec := recover(); rec != nil {
			log.Errorf("uncaught error in adding task logs event: %v", rec)
		}
	}()

	regex, ok := trigger.Condition[regexConditionKey].(string)
	if !ok {
		return fmt.Errorf(
			"expected webhook trigger to have regex in condition instead got %v", trigger.Condition)
	}

	p, err := generateTaskLogPayload(
		ctx, taskID, nodeName, regex, triggeringLog, trigger.Webhook.WebhookType)
	if err != nil {
		return fmt.Errorf("generating task logs event: %w", err)
	}

	needToWake := false
	if err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := db.Bun().NewInsert().Model(&webhookTaskLogTrigger{
			TaskID:    taskID,
			TriggerID: trigger.ID,
		}).On("CONFLICT (task_id, trigger_id) DO NOTHING").Exec(ctx)
		if err != nil {
			return fmt.Errorf("inserting task logs event trigger: %w", err)
		}
		if rowsAffected, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("getting rows affected for webhook task logs triggers: %w", err)
		} else if rowsAffected == 0 {
			return nil
		}

		if _, err := db.Bun().NewInsert().Model(&Event{
			Payload: p,
			URL:     trigger.Webhook.URL,
		}).Exec(ctx); err != nil {
			return fmt.Errorf("inserting task logs event trigger: %w", err)
		}

		needToWake = true
		return nil
	}); err != nil {
		return fmt.Errorf("adding webhook task log trigger event: %w", err)
	}

	if needToWake {
		singletonShipper.Wake()
	}

	return nil
}

func generateTaskLogPayload(
	ctx context.Context,
	taskID model.TaskID,
	nodeName,
	regex,
	triggeringLog string,
	wt WebhookType,
) ([]byte, error) {
	switch wt {
	case WebhookTypeDefault:
		p, err := json.Marshal(EventPayload{
			ID:        uuid.New(),
			Type:      TriggerTypeTaskLog,
			Timestamp: time.Now().Unix(),
			Condition: Condition{
				Regex: regex,
			},
			Data: EventData{
				TaskLog: &TaskLogPayload{
					TaskID:        taskID,
					NodeName:      nodeName,
					TriggeringLog: triggeringLog,
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("marshaling json for log pattern payload: %w", err)
		}

		return p, nil

	case WebhookTypeSlack:
		p, err := generateLogPatternSlackPayload(ctx, taskID, nodeName, regex, triggeringLog)
		if err != nil {
			return nil, err
		}
		return p, nil

	default:
		return nil, fmt.Errorf("unknown webhook type %+v while generating log pattern payload", wt)
	}
}

func generateLogPatternSlackPayload(
	ctx context.Context,
	taskID model.TaskID,
	nodeName,
	regex,
	triggeringLog string,
) ([]byte, error) {
	task, err := db.TaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	msg := ""
	if task.TaskType == model.TaskTypeTrial {
		trial, err := db.TrialByTaskID(ctx, taskID)
		if err != nil {
			return nil, err
		}
		msg = fmt.Sprintf(
			"Experiment ID `%d`, Trial ID `%d`, running on node `%s`, reported a log\n",
			trial.ExperimentID, trial.ID, nodeName) +
			fmt.Sprintf("```%s```\n", triggeringLog) +
			"This log matched the regex\n" +
			fmt.Sprintf("```%s```\n", regex)

		path := fmt.Sprintf("/det/experiments/%d/trials/%d/logs", trial.ExperimentID, trial.ID)
		if baseURL := conf.GetMasterConfig().Webhooks.BaseURL; baseURL != "" {
			msg += fmt.Sprintf("<%s%s | View full logs here>", baseURL, path)
		} else {
			msg += fmt.Sprintf("View full logs at %s", path)
		}
	} else {
		msg = fmt.Sprintf(
			"Task ID `%s`, task type `%s`, running on node `%s`, reported a log\n",
			taskID, task.TaskType, nodeName) +
			fmt.Sprintf("```%s```\n", triggeringLog) +
			"This log matched the regex\n" +
			fmt.Sprintf("```%s```\n", regex)
	}

	message, err := json.Marshal(SlackMessageBody{
		Blocks: []SlackBlock{
			{
				Type: "section",
				Text: SlackField{
					Type: "mrkdwn",
					Text: msg,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating slack payload: %w", err)
	}

	return message, nil
}

func generateEventPayload(
	ctx context.Context,
	wt WebhookType,
	e model.Experiment,
	activeConfig expconf.ExperimentConfig,
	expState model.State,
	tT TriggerType,
) ([]byte, error) {
	switch wt {
	case WebhookTypeDefault:
		pJSON, err := json.Marshal(EventPayload{
			ID:        uuid.New(),
			Type:      tT,
			Timestamp: time.Now().Unix(),
			Condition: Condition{
				State: expState,
			},
			Data: EventData{
				Experiment: experimentToWebhookPayload(e, activeConfig),
			},
		})
		if err != nil {
			return nil, err
		}
		return pJSON, nil
	case WebhookTypeSlack:
		slackJSON, err := generateSlackPayload(ctx, e, activeConfig)
		if err != nil {
			return nil, err
		}
		return slackJSON, nil
	default:
		panic(fmt.Errorf("unknown webhook type: %+v", wt))
	}
}

func generateSlackPayload(
	ctx context.Context, e model.Experiment, activeConfig expconf.ExperimentConfig,
) ([]byte, error) {
	var status string
	var eURL string
	var c string
	var mStatus string
	var projectID int
	var wID int
	var w *model.Workspace
	config := conf.GetMasterConfig()
	wName := activeConfig.Workspace() // TODO(!!!) this is incorrect on moves.
	pName := activeConfig.Project()
	webUIBaseURL := config.Webhooks.BaseURL
	baseURLIsSet := webUIBaseURL != ""
	if baseURLIsSet && wName != "" && pName != "" {
		ws, err := workspace.WorkspaceByName(ctx, wName)
		if err != nil {
			return nil, err
		}
		w = ws

		if w == nil {
			return nil, fmt.Errorf("unable to find workspace with name: %v", wName)
		}
		wID = w.ID

		pID, err := workspace.ProjectIDByName(ctx, wID, pName)
		if pID != nil {
			projectID = *pID
		}
		if err != nil {
			return nil, err
		}
	}

	if e.State == model.CompletedState {
		status = "Your experiment completed successfully 🎉"
		if baseURLIsSet {
			eURL = fmt.Sprintf("✅ <%v/det/experiments/%v/overview | %v (#%v)>",
				webUIBaseURL, e.ID, activeConfig.Name(), e.ID)
		} else {
			eURL = fmt.Sprintf("✅ %v (#%v)", activeConfig.Name(), e.ID)
		}
		c = "#13B670"
		mStatus = "Completed"
	} else {
		status = "Your experiment has stopped with errors"
		if baseURLIsSet {
			eURL = fmt.Sprintf("❌ <%v/det/experiments/%v/overview | %v (#%v)>",
				webUIBaseURL, e.ID, activeConfig.Name(), e.ID)
		} else {
			eURL = fmt.Sprintf("❌ %v (#%v)", activeConfig.Name(), e.ID)
		}
		c = "#DD5040"
		mStatus = "Errored"
	}
	hours := e.EndTime.Sub(e.StartTime).Hours()
	hours, m := math.Modf(hours)
	minutes := int(m * 60)
	duration := fmt.Sprintf("%vh %vmin", hours, minutes)
	expBlockFields := []SlackField{
		{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Status*: %v", mStatus),
		},
		{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Duration*: %v", duration),
		},
	}
	if wID != 0 && wName != "" && baseURLIsSet {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Workspace*: <%v/det/workspaces/%v/projects | %v>",
				webUIBaseURL, wID, wName),
		})
	} else if wName != "" {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Workspace*: %v", wName),
		})
	}
	if projectID != 0 && pName != "" && baseURLIsSet {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Project*: <%v/det/projects/%v | %v>",
				webUIBaseURL, projectID, pName),
		})
	} else if pName != "" {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Project*: %v", pName),
		})
	}
	experimentBlock := SlackBlock{
		Text: SlackField{
			Type: "mrkdwn",
			Text: eURL,
		},
		Type:   "section",
		Fields: &expBlockFields,
	}
	messageBlock := SlackBlock{
		Text: SlackField{
			Text: status,
			Type: "plain_text",
		},
		Type: "section",
	}
	attachment := SlackAttachment{
		Color:  c,
		Blocks: []SlackBlock{experimentBlock},
	}
	messageBody := SlackMessageBody{
		Blocks:      []SlackBlock{messageBlock},
		Attachments: &[]SlackAttachment{attachment},
	}

	message, err := json.Marshal(messageBody)
	if err != nil {
		return nil, fmt.Errorf("error creating slack payload: %w", err)
	}
	return message, nil
}

type eventBatch struct {
	tx       *bun.Tx
	events   []Event
	consumed bool
}

func (b *eventBatch) rollback() error {
	if !b.consumed {
		return b.tx.Rollback()
	}
	return nil
}

func (b *eventBatch) commit() error {
	b.consumed = true
	if err := b.tx.Commit(); err != nil {
		return fmt.Errorf("consuming event batch: %w", err)
	}
	return nil
}

func dequeueEvents(ctx context.Context, limit int) (*eventBatch, error) {
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	var events []Event
	if err = tx.NewRaw(`
DELETE FROM webhook_events_queue
USING ( SELECT * FROM webhook_events_queue LIMIT ? FOR UPDATE SKIP LOCKED ) q
WHERE q.id = webhook_events_queue.id RETURNING webhook_events_queue.*
`, limit).Scan(ctx, &events); err != nil {
		return nil, fmt.Errorf("scanning events: %w", err)
	}
	return &eventBatch{tx: &tx, events: events}, nil
}
