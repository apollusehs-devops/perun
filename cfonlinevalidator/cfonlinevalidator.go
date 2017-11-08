// Package cfonlinevalidator privides tools for online cloudformation template validation using AWS API.
package cfonlinevalidator

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/Appliscale/cftool/cflogger"
	"github.com/Appliscale/cftool/cfcontext"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-ini/ini"
	"os/user"
	"time"
	"io/ioutil"
	"errors"
)

const dateFormat = "2006-01-02 15:04:05 MST"

// Validate template and get URL for cost estimation.
func ValidateAndEstimateCosts(context *cfcontext.Context) bool {
	valid := false
	defer printResult(&valid, context.Logger)

	if *context.CliArguments.MFA {
		err := updateSessionToken(context.Config.Profile, context.Config.Region, context.Logger)
		if err != nil {
			context.Logger.Error(err.Error())
			return false
		}
	}

	session, err := createSession(&context.Config.Region, context.Config.Profile, context.Logger)
	if err != nil {
		context.Logger.Error(err.Error())
		return false
	}

	rawTemplate, err := ioutil.ReadFile(*context.CliArguments.TemplatePath)
	if err != nil {
		context.Logger.Error(err.Error())
		return false
	}

	template := string(rawTemplate)
	valid, err = isTemplateValid(session, &template)
	if err != nil {
		context.Logger.Error(err.Error())
		return false
	}

	estimateCosts(session, &template, context.Logger)

	return valid
}

func isTemplateValid(session *session.Session, template *string) (bool, error) {
	cfm := cloudformation.New(session)
	templateStruct := cloudformation.ValidateTemplateInput{
		TemplateBody: template,
	}
	_, error := cfm.ValidateTemplate(&templateStruct)
	if error != nil {
		return false, error
	}

	return true, nil
}

func estimateCosts(session *session.Session, template *string, logger *cflogger.Logger) {
	cfm := cloudformation.New(session)
	templateCostInput := cloudformation.EstimateTemplateCostInput{
		TemplateBody: template,
	}
	output, err := cfm.EstimateTemplateCost(&templateCostInput)

	if err != nil {
		logger.Error(err.Error())
		return
	}

	logger.Info("Costs estimation: " + *output.Url)
	/*resp, _ := http.Get(*output.Url)
	bytes, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("HTML:\n\n", string(bytes))*/
}

func createSession(region *string, profile string, logger *cflogger.Logger) (*session.Session, error) {
	logger.Info("Profile: " + profile)
	logger.Info("Region: " + *region)
	session, err := session.NewSessionWithOptions(
		session.Options{
			Config: aws.Config{
				Region: region,
			},
			Profile: profile,
		})
	if err != nil {
		return nil, err
	}

	return session, nil
}

func updateSessionToken(profile string, region string, logger *cflogger.Logger) error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	credentialsFilePath := user.HomeDir + "/.aws/credentials"
	cfg, err := ini.Load(credentialsFilePath)
	if err != nil {
		return err
	}

	section, err := cfg.GetSection(profile)
	if err != nil {
		section, err = cfg.NewSection(profile)
		if err != nil {
			return err
		}
	}

	profileLongTerm := profile + "-long-term"
	sectionLongTerm, err := cfg.GetSection(profileLongTerm)
	if err != nil {
		return err
	}

	sessionToken := section.Key("aws_session_token")
	expiration := section.Key("expiration")

	expirationDate, err := time.Parse(dateFormat, section.Key("expiration").Value())
	if err == nil {
		logger.Info("Session token will expire in " +
			time.Since(expirationDate).Truncate(time.Duration(1) * time.Second).String() +
			" (" + expirationDate.Truncate(time.Duration(1) * time.Second).Format(dateFormat) + ")")
	}

	mfaDevice := sectionLongTerm.Key("mfa_serial").Value()
	if mfaDevice == "" {
		return errors.New("There is no mfa_serial for the profile " + profileLongTerm)
	}

	if sessionToken.Value() == "" || expiration.Value() == "" || time.Since(expirationDate).Nanoseconds() > 0 {
		session, err := session.NewSessionWithOptions(
			session.Options{
				Config: aws.Config{
					Region: &region,
				},
				Profile: profileLongTerm,
			})
		if err != nil {
			return err
		}

		var tokenCode string
		err = logger.GetInput("MFA token code", &tokenCode)
		if err != nil {
			return err
		}

		var duration int64
		err = logger.GetInput("Duration", &duration)
		if err != nil {
			return err
		}

		stsSession := sts.New(session)
		newToken, err := stsSession.GetSessionToken(&sts.GetSessionTokenInput{
			DurationSeconds: &duration,
			SerialNumber:    aws.String(mfaDevice),
			TokenCode:       &tokenCode,
		})
		if err != nil {
			return err
		}

		section.Key("aws_access_key_id").SetValue(*newToken.Credentials.AccessKeyId)
		section.Key("aws_secret_access_key").SetValue(*newToken.Credentials.SecretAccessKey)
		sessionToken.SetValue(*newToken.Credentials.SessionToken)
		section.Key("expiration").SetValue(newToken.Credentials.Expiration.Format(dateFormat))

		cfg.SaveTo(credentialsFilePath)
	}

	return nil
}

func printResult(valid *bool, logger *cflogger.Logger) {
	if !*valid {
		logger.Info("Template is invalid!")
	} else {
		logger.Info("Template is valid!")
	}
}
