import { Breadcrumb } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getExperimentDetails, getProject, getTrialDetails, getWorkspace } from 'services/api';
import { ExperimentBase, Project, TrialDetails, Workspace } from 'types';
import { isEqual } from 'utils/data';
import handleError from 'utils/error';

import css from './BreadcrumbBar.module.scss';
import Link from './Link';
import WorkspaceIcon from './WorkspaceIcon';

interface Props {
  experiment?: ExperimentBase;
  id: number;
  project?: Project;
  trial?: TrialDetails;
  type: 'project' | 'experiment' | 'trial';
  workspace?: Workspace;
}

const BreadcrumbBar: React.FC<Props> = (
  {
    id, type, workspace: workspaceIn, project: projectIn,
    experiment: experimentIn, trial: trialIn,
  }: Props,
) => {
  const [ workspace, setWorkspace ] = useState<Workspace | undefined>(workspaceIn);
  const [ project, setProject ] = useState<Project | undefined>(projectIn);
  const [ experiment, setExperiment ] = useState<ExperimentBase | undefined>(experimentIn);
  const [ trial, setTrial ] = useState<TrialDetails | undefined>(trialIn);
  const [ canceler ] = useState(new AbortController());

  const fetchWorkspace = useCallback(async () => {
    if (!project?.workspaceId) return;
    try {
      const response = await getWorkspace({ id: project.workspaceId }, { signal: canceler.signal });
      setWorkspace(response);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch workspace.' });
    }
  }, [ canceler.signal, project?.workspaceId ]);

  const fetchProject = useCallback(async () => {
    if (type !== 'project' && experiment?.projectId === undefined) return;
    try {
      const response = await getProject(
        { id: type === 'project' ? id : experiment?.projectId ?? 1 },
        { signal: canceler.signal },
      );
      setProject(prev => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch project.' });
    }
  }, [ canceler.signal, experiment?.projectId, id, type ]);

  const fetchExperiment = useCallback(async () => {
    if (type !== 'experiment' && trial?.experimentId === undefined) return;
    try {
      const response = await getExperimentDetails(
        { id: type === 'experiment' ? id : trial?.experimentId ?? 1 },
        { signal: canceler.signal },
      );
      setExperiment(prev => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiment.' });
    }
  }, [ canceler.signal, id, trial?.experimentId, type ]);

  const fetchTrial = useCallback(async () => {
    if (type !== 'trial') return;
    try {
      const response = await getTrialDetails({ id }, { signal: canceler.signal });
      setTrial(prev => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch trial.' });
    }
  }, [ canceler.signal, id, type ]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([
      fetchProject(), fetchWorkspace(), fetchExperiment(), fetchTrial() ]);
  }, [ fetchProject, fetchWorkspace, fetchExperiment, fetchTrial ]);

  usePolling(fetchAll);

  useEffect(() => {
    fetchWorkspace();
  }, [ fetchWorkspace ]);

  useEffect(() => {
    fetchProject();
  }, [ fetchProject ]);

  useEffect(() => {
    fetchExperiment();
  }, [ fetchExperiment ]);

  useEffect(() => {
    fetchTrial();
  }, [ fetchTrial ]);

  return (
    <div className={css.base}>
      <Breadcrumb separator="">
        {experiment?.projectId !== 1 && (
          <>
            <Breadcrumb.Item>
              <Link path={project ? paths.workspaceDetails(project.workspaceId) : undefined}>
                <WorkspaceIcon name={workspace?.name} size={24} style={{ marginRight: 10 }} />
              </Link>
            </Breadcrumb.Item>
            <Breadcrumb.Item>
              <Link path={project ? paths.workspaceDetails(project.workspaceId) : undefined}>
                {workspace?.name ?? '...'}
              </Link>
            </Breadcrumb.Item>
            <Breadcrumb.Separator />
          </>
        )}
        <Breadcrumb.Item>
          <Link path={experiment ? paths.projectDetails(experiment.projectId) : undefined}>
            {project?.name ?? '...'}
          </Link>
        </Breadcrumb.Item>
        {(type === 'experiment' || type === 'trial') && (
          <>
            <Breadcrumb.Separator />
            <Breadcrumb.Item>
              <Link path={trial ? paths.experimentDetails(trial.experimentId) : undefined}>
                {experiment?.name ?? '...'}
              </Link>
            </Breadcrumb.Item>
          </>
        )}
        {type === 'trial' && (
          <>
            <Breadcrumb.Separator />
            <Breadcrumb.Item>
              {id ?? '...'}
            </Breadcrumb.Item>
          </>
        )}
      </Breadcrumb>
    </div>
  );
};

export default BreadcrumbBar;
