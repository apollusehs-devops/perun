package stack

import (
	"github.com/Appliscale/perun/context"
	"github.com/Appliscale/perun/mysession"
	"github.com/Appliscale/perun/progress"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func UpdateStack(context *context.Context) (err error) {
	template, stackName, err := getTemplateFromFile(context)
	if err != nil {
		return
	}
	templateStruct := updateStackInput(context, &template, &stackName)
	currentSession := mysession.InitializeSession(context)
	err = doUpdateStack(context, currentSession, templateStruct)
	return
}
func doUpdateStack(context *context.Context, currentSession *session.Session, updateStackInput cloudformation.UpdateStackInput) error {
	if *context.CliArguments.Progress {
		conn, remoteSinkError := progress.GetRemoteSink(context, currentSession)
		if remoteSinkError != nil {
			context.Logger.Error("Error getting remote sink configuration: " + remoteSinkError.Error())
			return remoteSinkError
		}
		updateStackInput.NotificationARNs = []*string{conn.TopicArn}
		updateError := updateStack(updateStackInput, currentSession)
		if updateError != nil {
			context.Logger.Error("Error updating stack: " + updateError.Error())
			return updateError
		}
		conn.MonitorStackQueue()
	} else {
		updateError := updateStack(updateStackInput, currentSession)
		if updateError != nil {
			context.Logger.Error("Error updating stack: " + updateError.Error())
			return updateError
		}
	}
	return nil
}

func updateStack(updateStackInput cloudformation.UpdateStackInput, session *session.Session) error {
	api := cloudformation.New(session)
	_, err := api.UpdateStack(&updateStackInput)
	return err
}

// This function gets template and  name of stack. It creates "CreateStackInput" structure.
func updateStackInput(context *context.Context, template *string, stackName *string) cloudformation.UpdateStackInput {
	params, err := getParameters(context)
	if err != nil {
		context.Logger.Error(err.Error())
		return cloudformation.UpdateStackInput{}
	}
	rawCapabilities := *context.CliArguments.Capabilities
	capabilities := make([]*string, len(rawCapabilities))
	for i, capability := range rawCapabilities {
		capabilities[i] = &capability
	}
	templateStruct := cloudformation.UpdateStackInput{
		Parameters:   params,
		TemplateBody: template,
		StackName:    stackName,
		Capabilities: capabilities,
	}
	return templateStruct
}