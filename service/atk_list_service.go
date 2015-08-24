/**
 * Copyright (c) 2015 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package service

import (
	"errors"
	"sort"
	"strings"

	"github.com/cloudfoundry/gosteno"
)

type AtkInstance struct {
	Name        string `json:"name"`
	Url         string `json:"url"`
	Guid        string `json:"guid"`
	ServiceGuid string `json:"service_guid"`
	State       string `json:"state"`
	SeInstance *AtkInstance `json:"scoring_engine"`
}


type AtkInstances struct {
	Instances         []AtkInstance `json:"instances"`
	ServicePlanGuid   string        `json:"service_plan_guid"`
	SeServicePlanGuid string      `json:"se_service_plan_guid"`
}

type ByName []AtkInstance

func (a ByName) Len() int { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func (a *AtkInstances) Append(another *AtkInstances) {
	if another.ServicePlanGuid != "" {
		a.ServicePlanGuid = another.ServicePlanGuid
		a.SeServicePlanGuid = another.SeServicePlanGuid
	}

	a.Instances = append(a.Instances, another.Instances...)
}

func (a *AtkInstances) Sort() {
	sort.Sort(ByName(a.Instances))}

func UuidToAppName(uuid string, label string) string {
	idx := strings.LastIndex(uuid, "-")
	return label + "-" + uuid[:idx]
}

type AtkListService struct {
	SpaceSummaryHelper SpaceSummaryHelper
	cloudController    CloudController
	logger          *gosteno.Logger
}

func NewAtkListService(cloudController CloudController, SpaceSummaryHelper SpaceSummaryHelper) *AtkListService {
	return &AtkListService{
		cloudController: cloudController,
		SpaceSummaryHelper: SpaceSummaryHelper,
		logger:          gosteno.NewLogger("atk_list_service"),
	}
}

func (p *AtkListService) getSpaceList(orgId string) ([]string, error) {
	spaces, err := p.cloudController.Spaces(orgId)
	if err != nil {
		return nil, err
	}

	return spaces.IdList(), nil
}

func (p *AtkListService) servicePlanId(Name string) (string, error) {
	services, err := p.cloudController.Services()
	if err != nil {
		return "", err
	}
	var servicePlansUrl string

	for _, r := range services.Resources {
		if r.Entity.Label == Name {
			servicePlansUrl = r.Entity.ServicePlansUrl
		}
	}

	if len(servicePlansUrl) == 0 {
		return "", errors.New("Service plans url is empty")
	}

	servicePlans, err := p.cloudController.ServicePlans(servicePlansUrl)
	if err != nil {
		return "", err
	}

	if len(servicePlans.Resources) == 0 {
		return "", errors.New("Could not find any service plan for: "+ Name)
	}

	return servicePlans.Resources[0].Metadata.Id, nil
}

func (p *AtkListService) getSpaceInstances(atkLabel string,
	seLabel string,
	serviceSearchString string,
	space string,
	instanceChan chan AtkInstances,
	errorChan chan error) {
	summary, err := p.cloudController.SpaceSummary(space)

	if err != nil {
		errorChan <- err
		return
	}
	atkInstanceList := p.getInstancesFromSpaceSummary(atkLabel, seLabel, serviceSearchString, summary);

	atkPlan, err := p.servicePlanId(atkLabel)

	if err != nil {
		p.logger.Warn("Failed to fetch service plan for label: " + atkLabel)
	}

	sePlan, err := p.servicePlanId(seLabel)
	if err != nil {
		p.logger.Warn("Failed to fetch service plan for label: " + seLabel)
	}

	instanceChan <-AtkInstances{atkInstanceList, atkPlan, sePlan}
}

func (p *AtkListService) getInstancesFromSpaceSummary(atkLabel string,
	seLabel string,
	serviceSearchString string,
	summary *SpaceSummary) []AtkInstance {
	apps := make(map[string]Application)
	for _, a := range summary.Apps {
		apps[a.Name] = a
	}

	p.logger.Info("Service search string : " + serviceSearchString)

	seMap := p.SpaceSummaryHelper.getMapOfAppsByService(seLabel, serviceSearchString, summary, apps)
	atkMap := p.SpaceSummaryHelper.getMapOfAppsByService(atkLabel, serviceSearchString, summary, apps)

	instances := make([]AtkInstance, len(summary.Services))

	j := 0
	for commonService, atk := range atkMap {
		se := seMap[commonService]
		atk.SeInstance = &se
		instances[j] = atk
		j++
	}
	return instances[:j]
}


func (p *AtkListService) GetAllInstances(atkLabel string, seLabel string, commonService string, orgId string) (*AtkInstances, error) {
	spaceList, err := p.getSpaceList(orgId)
	if err != nil {
		return nil, err
	}

	instanceChan := make(chan AtkInstances)
	errorChan := make(chan error)

	for _, s := range spaceList {
		go p.getSpaceInstances(atkLabel, seLabel, commonService, s, instanceChan, errorChan)
	}

	atkInstances := AtkInstances{}
	for _, _ = range spaceList {
		select {
		case spaceInstances := <-instanceChan:
			atkInstances.Append(&spaceInstances)
		case err = <-errorChan:
			p.logger.Warn(err.Error())
		}
	}

	return &atkInstances, nil
}
