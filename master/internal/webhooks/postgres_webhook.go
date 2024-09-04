package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	conf "github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/google/uuid"
)

type regexTriggers struct {
	re                 *regexp.Regexp
	triggerIDToTrigger map[TriggerID]*Trigger
}

// WebhookManager manages webhooks.
type WebhookManager struct {
	mu                 sync.RWMutex
	regexToTriggers    map[string]regexTriggers
	expToWebhookConfig map[int]*expconf.WebhooksConfigV0
}

// CustomTriggerData is the data for custom trigger.
type CustomTriggerData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Level       string `json:"level"`
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
		regexToTriggers:    make(map[string]regexTriggers),
		expToWebhookConfig: make(map[int]*expconf.WebhooksConfigV0),
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

func (l *WebhookManager) getWebhookConfig(ctx context.Context, expID *int) (*expconf.WebhooksConfigV0, error) {
	if expID == nil {
		return nil, nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	if config, ok := l.expToWebhookConfig[*expID]; ok {
		return config, nil
	}
	var expConfigBytes []byte
	err := db.Bun().NewSelect().Table("experiments").Column("config").Where("id = ?", *expID).Scan(ctx, &expConfigBytes)
	if err != nil {
		return nil, err
	}
	expConfig, err := expconf.ParseAnyExperimentConfigYAML(expConfigBytes)
	if err != nil {
		return nil, err
	}
	if integration := schemas.WithDefaults(expConfig).Integrations(); integration != nil {
		l.expToWebhookConfig[*expID] = integration.Webhooks
		return integration.Webhooks, nil
	}
	l.expToWebhookConfig[*expID] = nil
	return nil, nil
}

func matchWebhook(t *Trigger, config *expconf.WebhooksConfigV0, workspaceID int32, expID *int) bool {
	if config != nil && config.Exclude {
		return false
	}
	// Global webhooks (no workspace ID and not specific) trigger for all tasks
	if t.Webhook.Mode == WebhookModeSpecific || t.Webhook.WorkspaceID != nil {
		if t.Webhook.WorkspaceID != nil {
			// Webhook is workspace level instead of global level
			if *t.Webhook.WorkspaceID != workspaceID {
				// Skip webhook from other workspaces
				return false
			}
		}
		if t.Webhook.Mode == WebhookModeSpecific {
			// Skip specific webhook if task is not an experiment
			if expID == nil || config == nil {
				return false
			}
			// For webhook with mode specific, only proceed for experiments with matching config
			if config.WebhookID != nil && slices.Contains(*config.WebhookID, int(t.Webhook.ID)) {
				return true
			}
			if config.WebhookName != nil && slices.Contains(*config.WebhookName, t.Webhook.Name) {
				return true
			}
			return false
		}
	}
	return true
}

func (l *WebhookManager) scanLogs(
	ctx context.Context, logs []*model.TaskLog, workspaceID model.AccessScopeID, expID *int,
) error {
	if len(logs) == 0 {
		return nil
	}

	config, err := l.getWebhookConfig(ctx, expID)
	if err != nil {
		return err
	}

	if config != nil && config.Exclude {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, log := range logs {
		if log.AgentID == nil {
			return fmt.Errorf("AgentID must be non nil to trigger webhooks in logs")
		}

		for _, cacheItem := range l.regexToTriggers {
			// One of the trial logs prints expconf which has the regex pattern.
			// We skip monitoring this line.
			if logpattern.ExpconfigCompiledRegex.MatchString(log.Log) {
				continue
			}
			if cacheItem.re.MatchString(log.Log) {
				for _, t := range cacheItem.triggerIDToTrigger {
					if matchWebhook(t, config, int32(workspaceID), expID) {
						if err := addTaskLogEvent(ctx,
							model.TaskID(log.TaskID), *log.AgentID, log.Log, t); err != nil {
							return err
						}
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

func handleCustomTriggerData(ctx context.Context, data CustomTriggerData, experimentID int, trialID *int) error {
	var m struct {
		bun.BaseModel `bun:"table:experiments"`
		model.Experiment
		ConfigBytes []byte `bun:"config"`
	}
	err := db.Bun().NewSelect().Model(&m).ExcludeColumn("username").Where("id = ?", experimentID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("error getting experiment from id %d: %w", experimentID, err)
	}

	activeConfig, err := expconf.ParseAnyExperimentConfigYAML(m.ConfigBytes)
	if err != nil {
		return fmt.Errorf("error parsing experiment config: %w", err)
	}
	integrationsConfig := activeConfig.Integrations()
	if integrationsConfig == nil {
		return nil
	}
	webhookConfig := integrationsConfig.Webhooks
	if webhookConfig == nil || webhookConfig.Exclude {
		return nil
	}

	workspaceID, err := experiment.GetWorkspaceFromExperiment(ctx, &m.Experiment)
	if err != nil {
		return fmt.Errorf("error getting workspace from experiment : %w", err)
	}

	var es []Event
	if webhookConfig.WebhookID != nil {
		for _, webhookID := range *webhookConfig.WebhookID {
			webhook, err := GetWebhook(ctx, webhookID)
			if err != nil {
				return fmt.Errorf("error getting webhook from id %d: %w", webhookID, err)
			}
			err = generateEventForCustomTrigger(
				ctx, &es, webhook.Triggers, webhook.WebhookType, webhook.URL, m.Experiment, activeConfig, data, trialID)
			if err != nil {
				return fmt.Errorf("error genrating event %d %+v: %w", webhookID, webhook, err)
			}
		}
	}
	if webhookConfig.WebhookName != nil {
		for _, webhookName := range *webhookConfig.WebhookName {
			webhook, err := getWebhookByName(ctx, webhookName, int(workspaceID))
			if err != nil {
				return fmt.Errorf("error getting webhook from name %s: %w", webhookName, err)
			}
			if webhook == nil {
				continue
			}
			err = generateEventForCustomTrigger(
				ctx, &es, webhook.Triggers, webhook.WebhookType, webhook.URL, m.Experiment, activeConfig, data, trialID)
			if err != nil {
				return fmt.Errorf("error genrating event %s %+v: %w", webhookName, webhook, err)
			}
		}
	}

	if len(es) == 0 {
		return nil
	}

	if _, err := db.Bun().NewInsert().Model(&es).Exec(ctx); err != nil {
		return fmt.Errorf("report experiment state changed inserting event trigger: %w", err)
	}

	singletonShipper.Wake()
	return nil
}

func generateEventForCustomTrigger(
	ctx context.Context,
	es *[]Event,
	triggers Triggers,
	webhookType WebhookType,
	webhookURL string,
	e model.Experiment,
	activeConfig expconf.ExperimentConfig,
	data CustomTriggerData,
	trialID *int,
) error {
	for _, t := range triggers {
		if t.TriggerType != TriggerTypeCustom {
			continue
		}
		p, err := generateEventPayload(
			ctx, webhookType, e, activeConfig, e.State, TriggerTypeCustom, &data, trialID,
		)
		if err != nil {
			return fmt.Errorf("error generating event payload: %w", err)
		}
		*es = append(*es, Event{Payload: p, URL: webhookURL})
	}
	return nil
}

func getWebhookByName(ctx context.Context, webhookName string, workspaceID int) (*Webhook, error) {
	webhook := Webhook{}
	err := db.Bun().NewSelect().
		Model(&webhook).
		Relation("Triggers").
		Where("name = ?", webhookName).
		Where("workspace_id = ?", workspaceID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &webhook, nil
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

// getWebhooks returns all global webhooks from the DB
// and all webhooks whose scopes are in workspaceIDs.
// workspaceIDs being nil gets only the globally scoped webhooks.
func getWebhooks(ctx context.Context, workspaceIDs *[]int32) (Webhooks, error) {
	webhooks := Webhooks{}
	q := db.Bun().NewSelect().
		Model(&webhooks).
		Relation("Triggers").
		Where("workspace_id is NULL")
	if workspaceIDs != nil {
		q.WhereOr("workspace_id IN (?)", bun.In(*workspaceIDs))
	}
	err := q.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

func getWorkspace(ctx context.Context, workspaceID int32) (*model.Workspace, error) {
	var workspace model.Workspace
	err := db.Bun().NewSelect().Model(&workspace).Where("id = ?", workspaceID).Scan(ctx)
	return &workspace, err
}

// ReportExperimentStateChanged adds webhook events to the queue.
// This function assumes ID presents in e.
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

	workspaceID, err := experiment.GetWorkspaceFromExperiment(ctx, &e)
	if err != nil {
		return fmt.Errorf("get workspace id from experiment %d: %w", e.ID, err)
	}
	var webhookConfig *expconf.WebhooksConfigV0
	if activeConfig.Integrations() != nil {
		webhookConfig = activeConfig.Integrations().Webhooks
	}

	var es []Event
	for _, t := range ts {
		if !matchWebhook(&t, webhookConfig, workspaceID, ptrs.Ptr(e.ID)) {
			continue
		}
		p, err := generateEventPayload(
			ctx, t.Webhook.WebhookType, e, activeConfig, e.State, TriggerTypeStateChange, nil, nil,
		)
		if err != nil {
			return fmt.Errorf("error generating event payload: %w", err)
		}
		es = append(es, Event{Payload: p, URL: t.Webhook.URL})
	}
	if len(es) == 0 {
		return nil
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
	eventData *CustomTriggerData, trialID *int,
) ([]byte, error) {
	switch wt {
	case WebhookTypeDefault:
		experiment := experimentToWebhookPayload(e, activeConfig)
		if trialID != nil && *trialID > 0 {
			experiment.TrialID = *trialID
		}
		pJSON, err := json.Marshal(EventPayload{
			ID:        uuid.New(),
			Type:      tT,
			Timestamp: time.Now().Unix(),
			Condition: Condition{
				State: expState,
			},
			Data: EventData{
				Experiment: experiment,
				CustomData: eventData,
			},
		})
		if err != nil {
			return nil, err
		}
		return pJSON, nil
	case WebhookTypeSlack:
		slackJSON, err := generateSlackPayload(ctx, e, activeConfig, eventData, trialID)
		if err != nil {
			return nil, err
		}
		return slackJSON, nil
	default:
		panic(fmt.Errorf("unknown webhook type: %+v", wt))
	}
}

func generateSlackPayload(
	ctx context.Context, e model.Experiment,
	activeConfig expconf.ExperimentConfig, eventData *CustomTriggerData, trialID *int,
) ([]byte, error) {
	var status string
	var eURL string
	var mStatus string
	var projectID int
	var wID int
	var w *model.Workspace
	c := "#13B670"
	config := conf.GetMasterConfig()
	wName := activeConfig.Workspace() // TODO(ET-288): This is incorrect on moves.
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

		pID, err := project.ProjectIDByName(ctx, wID, pName)
		if pID != nil {
			projectID = *pID
		}
		if err != nil {
			return nil, err
		}
	}

	switch e.State {
	case model.CompletedState:
		status = "Your experiment completed successfully 🎉"
		if baseURLIsSet {
			eURL = fmt.Sprintf("✅ <%v/det/experiments/%v/overview | %v (#%v)>",
				webUIBaseURL, e.ID, activeConfig.Name(), e.ID)
		} else {
			eURL = fmt.Sprintf("✅ %v (#%v)", activeConfig.Name(), e.ID)
		}
		mStatus = "Completed"
	case model.ErrorState:
		status = "Your experiment has stopped with errors"
		if baseURLIsSet {
			eURL = fmt.Sprintf("❌ <%v/det/experiments/%v/overview | %v (#%v)>",
				webUIBaseURL, e.ID, activeConfig.Name(), e.ID)
		} else {
			eURL = fmt.Sprintf("❌ %v (#%v)", activeConfig.Name(), e.ID)
		}
		c = "#DD5040"
		mStatus = "Errored"
	default:
		status = fmt.Sprintf("The status of your experiment is %s", e.State)
		if baseURLIsSet {
			eURL = fmt.Sprintf("<%v/det/experiments/%v/overview | %v (#%v)>",
				webUIBaseURL, e.ID, activeConfig.Name(), e.ID)
		} else {
			eURL = fmt.Sprintf("%v (#%v)", activeConfig.Name(), e.ID)
		}
		mStatus = string(e.State)
	}

	endTime := time.Now()
	if e.EndTime != nil {
		endTime = *e.EndTime
	}
	hours := endTime.Sub(e.StartTime).Hours()
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
	if trialID != nil && *trialID > 0 {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Trial ID*: %d", *trialID),
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
	if eventData != nil {
		eventDataFields := []SlackField{
			{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Level*: %v", eventData.Level),
			},
			{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Title*: %v", eventData.Title),
			},
			{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Description*: %v", eventData.Description),
			},
		}
		messageBlock = SlackBlock{
			Text: SlackField{
				Text: "Event Data",
				Type: "plain_text",
			},
			Type:   "section",
			Fields: &eventDataFields,
		}
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
