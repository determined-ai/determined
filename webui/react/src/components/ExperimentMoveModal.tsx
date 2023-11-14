import { Typography } from 'antd';
import Form from 'hew/Form';
import Icon from 'hew/Icon';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useEffect, useId, useState } from 'react';

import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { moveExperiments } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import projectStore from 'stores/projects';
import workspaceStore from 'stores/workspaces';
import { Project } from 'types';
import handleError from 'utils/error';
import { pluralizer } from 'utils/string';

const FORM_ID = 'move-experiment-form';

type FormInputs = {
  projectId?: number;
  workspaceId?: number;
};

interface Props {
  excludedExperimentIds?: Set<number>;
  experimentIds: number[];
  filters?: V1BulkExperimentFilters;
  onSubmit?: (successfulIds?: number[]) => void;
  sourceProjectId?: number;
  sourceWorkspaceId?: number;
}

const ExperimentMoveModalComponent: React.FC<Props> = ({
  excludedExperimentIds,
  experimentIds,
  filters,
  onSubmit,
  sourceProjectId,
  sourceWorkspaceId,
}: Props) => {
  const idPrefix = useId();
  const { openToast } = useToast();
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

  useEffect(() => {
    if (workspaceId !== undefined) {
      projectStore.fetch(workspaceId, undefined, true);
    }
  }, [workspaceId]);

  const handleSubmit = async () => {
    if (workspaceId === sourceWorkspaceId && projectId === sourceProjectId) {
      openToast({ title: 'No changes to save.' });
      return;
    }
    const values = await form.validateFields();
    const projId = values.projectId ?? 1;

    if (excludedExperimentIds?.size) {
      filters = { ...filters, excludedExperimentIds: Array.from(excludedExperimentIds) };
    }

    const results = await moveExperiments({
      destinationProjectId: projId,
      experimentIds,
      filters,
    });

    onSubmit?.(results.successful);

    const numSuccesses = results.successful.length;
    const numFailures = results.failed.length;

    const destinationProjectName =
      Loadable.getOrElse([], loadableProjects).find((p) => p.id === projId)?.name ?? '';

    if (numSuccesses === 0 && numFailures === 0) {
      openToast({
        description: 'No selected experiments were eligible for moving',
        title: 'No eligible experiments',
      });
    } else if (numFailures === 0) {
      openToast({
        closeable: true,
        description: `${results.successful.length} experiments moved to project ${destinationProjectName}`,
        link: <Link path={paths.projectDetails(projId)}>View Project</Link>,
        title: 'Move Success',
      });
    } else if (numSuccesses === 0) {
      openToast({
        description: `Unable to move ${numFailures} experiments`,
        severity: 'Warning',
        title: 'Move Failure',
      });
    } else {
      openToast({
        closeable: true,
        description: `${numFailures} out of ${
          numFailures + numSuccesses
        } eligible experiments failed to move
      to project ${destinationProjectName}`,
        link: <Link path={paths.projectDetails(projId)}>View Project</Link>,
        severity: 'Warning',
        title: 'Partial Move Failure',
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
        form: idPrefix + FORM_ID,
        handleError,
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
      <Form form={form} id={idPrefix + FORM_ID} layout="vertical">
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
                    {workspace.archived && <Icon name="archive" title="Archived" />}
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
              Failed: () => null, // Inform the user if this fails to load
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
                        {project.archived && <Icon name="archive" title="Archived" />}
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
