import { Typography } from 'antd';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useState } from 'react';

import Form from 'components/kit/Form';
import Icon from 'components/kit/Icon';
import { Modal } from 'components/kit/Modal';
import Select, { Option } from 'components/kit/Select';
import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { moveExperiments } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { pluralizer } from 'shared/utils/string';
import projectStore from 'stores/projects';
import workspaceStore from 'stores/workspaces';
import { Project } from 'types';
import { message, notification } from 'utils/dialogApi';
import { Loadable } from 'utils/loadable';

type FormInputs = {
  projectId?: number;
  workspaceId?: number;
};

interface Props {
  onSubmit?: (successfulIds?: number[]) => void;
  experimentIds: number[];
  filters?: V1BulkExperimentFilters;
  sourceProjectId?: number;
  sourceWorkspaceId?: number;
}

const ExperimentMoveModalComponent: React.FC<Props> = ({
  onSubmit,
  experimentIds,
  filters,
  sourceProjectId,
  sourceWorkspaceId,
}: Props) => {
  const [disabled, setDisabled] = useState<boolean>(true);
  const [form] = Form.useForm<FormInputs>();
  const workspaceId = Form.useWatch('workspaceId', form);
  const projectId = Form.useWatch('projectId', form);

  useEffect(() => {
    setDisabled(workspaceId !== 1 && !projectId);
  }, [workspaceId, projectId, sourceProjectId, sourceWorkspaceId]);

  const { canMoveExperimentsTo } = usePermissions();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.unarchived)).filter((w) =>
    canMoveExperimentsTo({ destination: { id: w.id } }),
  );
  const loadableProjects: Loadable<Project[]> = useObservable(
    projectStore.getProjectsByWorkspace(workspaceId),
  );

  useEffect(() => workspaceStore.fetch(), []);

  useEffect(
    () => (workspaceId === undefined ? undefined : projectStore.fetch(workspaceId)),
    [workspaceId],
  );

  const closeNotification = useCallback(() => notification.destroy(), []);

  const handleSubmit = async () => {
    if (workspaceId === sourceWorkspaceId && projectId === sourceProjectId) {
      message.info('No changes to save.');
      return;
    }
    const values = await form.validateFields();
    const projId = values.projectId ?? 1;

    const results = await moveExperiments({ destinationProjectId: projId, experimentIds, filters });

    onSubmit?.(results.successful);

    const numSuccesses = results.successful.length;
    const numFailures = results.failed.length;

    const destinationProjectName =
      Loadable.getOrElse([], loadableProjects).find((p) => p.id === projId)?.name ?? '';

    if (numSuccesses === 0 && numFailures === 0) {
      notification.open({
        description: 'No selected experiments were eligible for moving',
        message: 'No eligible experiments',
      });
    } else if (numFailures === 0) {
      notification.open({
        btn: null,
        description: (
          <div onClick={closeNotification}>
            <p>
              {results.successful.length} experiments moved to project {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(projId)}>View Project</Link>
          </div>
        ),
        message: 'Move Success',
      });
    } else if (numSuccesses === 0) {
      notification.warning({
        description: `Unable to move ${numFailures} experiments`,
        message: 'Move Failure',
      });
    } else {
      notification.warning({
        description: (
          <div onClick={closeNotification}>
            <p>
              {numFailures} out of {numFailures + numSuccesses} eligible experiments failed to move
              to project {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(projId)}>View Project</Link>
          </div>
        ),
        key: 'move-notification',
        message: 'Partial Move Failure',
      });
    }
    form.resetFields();
  };

  useEffect(() => {
    form.setFieldValue('projectId', sourceProjectId);
    form.setFieldValue('workspaceId', sourceWorkspaceId ?? 1);
  }, [form, sourceProjectId, sourceWorkspaceId]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        handler: handleSubmit,
        text:
          filters !== undefined
            ? 'Move Experiments'
            : `Move ${pluralizer(experimentIds.length, 'Experiment')}`,
      }}
      title={
        filters !== undefined
          ? 'Move Experiments'
          : `Move ${pluralizer(experimentIds.length, 'Experiment')}`
      }>
      <Form form={form} layout="vertical">
        <Form.Item
          label="Workspace"
          name="workspaceId"
          rules={[{ message: 'Workspace is required', required: true }]}>
          <Select
            filterOption={(input, option) =>
              (option?.title?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
            }
            id="workspace"
            placeholder="Select a destination workspace."
            onChange={() => form.resetFields(['projectId'])}>
            {workspaces.map((workspace) => {
              return (
                <Option
                  disabled={workspace.archived}
                  key={workspace.id}
                  title={workspace.name}
                  value={workspace.id}>
                  <div>
                    <Typography.Text ellipsis={true}>{workspace.name}</Typography.Text>
                    {workspace.archived && <Icon name="archive" />}
                  </div>
                </Option>
              );
            })}
          </Select>
        </Form.Item>
        {workspaceId && workspaceId !== 1 && (
          <Form.Item
            label="Project"
            name="projectId"
            rules={[{ message: 'Project is required', required: true }]}>
            {Loadable.match(loadableProjects, {
              Loaded: (loadableProjects) => (
                <Select
                  filterOption={(input, option) =>
                    (option?.title?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
                  }
                  placeholder="Select a destination project.">
                  {loadableProjects.map((project) => (
                    <Option
                      disabled={project.archived}
                      key={project.id}
                      title={project.name}
                      value={project.id}>
                      <div>
                        <Typography.Text ellipsis={true}>{project.name}</Typography.Text>
                        {project.archived && <Icon name="archive" />}
                      </div>
                    </Option>
                  ))}
                </Select>
              ),
              NotLoaded: () => <Spinner center spinning />,
            })}
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
};

export default ExperimentMoveModalComponent;
