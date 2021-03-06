package nudge

import (
	"net/http"
	"time"

	"github.com/spf13/viper"
	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	"github.com/bitmark-inc/autonomy-api/background"
	"github.com/bitmark-inc/autonomy-api/external/cadence"
	"github.com/bitmark-inc/autonomy-api/external/onesignal"
	"github.com/bitmark-inc/autonomy-api/store"
)

const TaskListName = "autonomy-nudge-tasks"

type NudgeWorker struct {
	domain             string
	mongo              store.MongoStore
	notificationCenter background.NotificationCenter
}

func NewNudgeWorker(domain string, mongo store.MongoStore) *NudgeWorker {
	o := onesignal.NewClient(&http.Client{
		Timeout: 15 * time.Second,
	})

	return &NudgeWorker{
		domain:             domain,
		mongo:              mongo,
		notificationCenter: background.NewOnesignalNotificationCenter(viper.GetString("onesignal.appid"), o),
	}
}

func (n *NudgeWorker) Register() {
	workflow.RegisterWithOptions(n.SymptomFollowUpNudgeWorkflow, workflow.RegisterOptions{Name: "SymptomFollowUpNudgeWorkflow"})
	workflow.RegisterWithOptions(n.NotifySymptomSpikeWorkflow, workflow.RegisterOptions{Name: "NotifySymptomSpikeWorkflow"})
	workflow.RegisterWithOptions(n.NotifyBehaviorOnEnteringRiskAreaWorkflow, workflow.RegisterOptions{Name: "NotifyBehaviorOnEnteringRiskAreaWorkflow"})
	workflow.RegisterWithOptions(n.AccountSelfReportedHighRiskFollowUpWorkflow, workflow.RegisterOptions{Name: "AccountSelfReportedHighRiskFollowUpWorkflow"})
	workflow.RegisterWithOptions(n.NotifyBehaviorFollowUpOnEnteringSymptomSpikeAreaWorkflow, workflow.RegisterOptions{Name: "NotifyBehaviorFollowUpOnEnteringSymptomSpikeAreaWorkflow"})

	activity.RegisterWithOptions(n.SymptomsNeedFollowUpActivity, activity.RegisterOptions{Name: "SymptomsNeedFollowUpActivity"})
	activity.RegisterWithOptions(n.NotifySymptomFollowUpActivity, activity.RegisterOptions{Name: "NotifySymptomFollowUpActivity"})
	activity.RegisterWithOptions(n.NotifySymptomSpikeActivity, activity.RegisterOptions{Name: "NotifySymptomSpikeActivity"})
	activity.RegisterWithOptions(n.NotifyBehaviorNudgeActivity, activity.RegisterOptions{Name: "NotifyBehaviorNudgeActivity"})
	activity.RegisterWithOptions(n.GetNotificationReceiverActivity, activity.RegisterOptions{Name: "GetNotificationReceiverActivity"})
	activity.RegisterWithOptions(n.CheckSelfHasHighRiskSymptomsAndNeedToFollowUpActivity, activity.RegisterOptions{Name: "HighRiskAccountFollowUpActivity"})
	activity.RegisterWithOptions(n.NotifyBehaviorFollowUpWhenSelfIsInHighRiskActivity, activity.RegisterOptions{Name: "NotifyBehaviorFollowUpActivity"})
}

func (n *NudgeWorker) Start(service workflowserviceclient.Interface, logger *zap.Logger) {
	// TaskListName identifies set of client workflows, activities, and workers.
	// It could be your group or client or application name.
	workerOptions := worker.Options{
		Logger:        logger,
		MetricsScope:  tally.NewTestScope(TaskListName, map[string]string{}),
		DataConverter: cadence.NewMsgPackDataConverter(),
	}

	worker := worker.New(
		service,
		n.domain,
		TaskListName,
		workerOptions)

	if err := worker.Start(); err != nil {
		panic("Failed to start worker")
	}

	logger.Info("Started Worker.", zap.String("worker", TaskListName))

	select {}
}
