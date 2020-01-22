package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/glassechidna/awsctx/service/elbv2ctx"
	"github.com/pkg/errors"
)

func main() {
	sess := session.Must(session.NewSession())
	api := elbv2ctx.New(elbv2.New(sess), nil)
	handler := lambda.NewHandler(New(api).Handle)
	handler = &logger{handler}
	lambda.StartHandler(handler)
}

type handler struct {
	api elbv2ctx.ELBV2
}

func New(api elbv2ctx.ELBV2) *handler {
	return &handler{api: api}
}

func (h *handler) Handle(ctx context.Context, input *MacroInput) (*MacroOutput, error) {
	tf := &templateFragment{fragment: input.Fragment}
	v := &myvisitor{}

	err := tf.Visit(ctx, v)
	if err != nil {
		return nil, err
	}

	err = h.handleAssociation(ctx, tf, v.props)
	if err != nil {
		return nil, err
	}

	return &MacroOutput{
		Status:    MacroOutputStatusSuccess,
		RequestId: input.RequestId,
		Fragment:  tf.fragment,
	}, nil
}

func (h *handler) handleAssociation(ctx context.Context, tf *templateFragment, props []albEventProperties) error {
	seen := map[string]bool{}

	for _, prop := range props {
		if len(prop.ListenerArn) == 0 || prop.VpcConfig != nil {
			return errors.New("TODO: support vpc config")
		}

		tgName := fmt.Sprintf("%sAlbTargetGroup", prop.Resource)
		pjName := fmt.Sprintf("%sAlbPermission", prop.Resource)
		lrName := fmt.Sprintf("%sAlb%sListenerRule", prop.Resource, prop.EventName)

		targetName := prop.Resource
		if len(prop.Alias) > 0 {
			targetName += ".Alias"
		}

		if _, found := seen[prop.Resource]; !found {
			seen[prop.Resource] = true

			tg := targetGroupJson(prop.Resource, targetName, prop.Tags)
			err := tf.PutResource(tgName, tg)
			if err != nil {
				return err
			}

			pj := permissionJson(targetName)
			err = tf.PutResource(pjName, pj)
			if err != nil {
				return err
			}
		}

		conds := convertConditions(prop.Conditions)
		//rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		//priority := 40_000 + rnd.Intn(10_000)
		//priority := calculatePriority(conds)
		priority := prop.Priority
		if priority == 0 {
			priority = 50_000
		}

		forward := ActionProperty{
			Type:           elbv2.ActionTypeEnumForward,
			TargetGroupArn: &stringOrRef{Ref: tgName},
		}

		rule := rule{
			Priority:    priority,
			ListenerArn: prop.ListenerArn,
			Conditions:  conds,
			Actions:     []ActionProperty{forward},
		}

		if prop.Oidc != nil {
			auth, err := oidcAction(prop.Oidc)
			if err != nil {
				return err
			}

			forward.Order = 2
			rule.Actions = []ActionProperty{*auth, forward}
		}

		lr := listenerRuleJson(rule)
		err := tf.PutResource(lrName, lr)
		if err != nil {
			return err
		}
	}

	return nil
}
