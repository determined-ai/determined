import Form from 'hew/Form';
import Icon from 'hew/Icon';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import { Label } from 'hew/Typography';
import { Loadable } from 'hew/utils/loadable';
import { List } from 'immutable';
import { useObservable } from 'micro-observables';
import React, { useEffect, useId } from 'react';

import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { moveRuns } from 'services/api';
import projectStore from 'stores/projects';
import workspaceStore from 'stores/workspaces';
import { FlatRun, Project } from 'types';
import handleError from 'utils/error';
import { pluralizer } from 'utils/string';

const FORM_ID = 'move-flat-run-form';

type FormInputs = {
  projectId?: number;
  workspaceId?: number;
};

interface Props {
  flatRuns: Readonly<FlatRun>[];
  onSubmit?: (successfulIds?: number[]) => void;
  sourceProjectId: number;
  sourceWorkspaceId?: number;
}

const FlatRunMoveModalComponent: React.FC<Props> = ({
  flatRuns,
  onSubmit,
  sourceProjectId,
  sourceWorkspaceId,
}: Props) => {
  const idPrefix = useId();
  const { openToast } = useToast();
  const [form] = Form.useForm<FormInputs>();
  const workspaceId = Form.useWatch('workspaceId', form);
  const projectId = Form.useWatch('projectId', form);

  const { canMoveExperimentsTo } = usePermissions();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.unarchived)).filter((w) =>
    canMoveExperimentsTo({ destination: { id: w.id } }),
  );
  const loadableProjects: Loadable<List<Project>> = useObservable(
    projectStore.getProjectsByWorkspace(workspaceId),
  );

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

    const results = await moveRuns({
      destinationProjectId: projId,
      runIds: flatRuns.map((flatRun) => flatRun.id),
      sourceProjectId,
    });

    onSubmit?.(results.successful);

    const numSuccesses = results.successful.length;
    const numFailures = results.failed.length;

    const destinationProjectName =
      Loadable.getOrElse(List<Project>(), loadableProjects).find((p) => p.id === projId)?.name ??
      '';

    if (numSuccesses === 0 && numFailures === 0) {
      openToast({
        description: 'No selected runs were eligible for moving',
        title: 'No eligible runs',
      });
    } else if (numFailures === 0) {
      openToast({
        closeable: true,
        description: `${results.successful.length} runs moved to project ${destinationProjectName}`,
        link: <Link path={paths.projectDetails(projId)}>View Project</Link>,
        title: 'Move Success',
      });
    } else if (numSuccesses === 0) {
      openToast({
        description: `Unable to move ${numFailures} runs`,
        severity: 'Warning',
        title: 'Move Failure',
      });
    } else {
      openToast({
        closeable: true,
        description: `${numFailures} out of ${numFailures + numSuccesses} eligible runs failed to move to project ${destinationProjectName}`,
        link: <Link path={paths.projectDetails(projId)}>View Project</Link>,
        severity: 'Warning',
        title: 'Partial Move Failure',
      });
    }
    form.resetFields();
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: workspaceId !== 1 && !projectId,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: `Move ${pluralizer(flatRuns.length, 'Runs')}`,
      }}
      title={`Move ${pluralizer(flatRuns.length, 'Runs')}`}>
      <Form form={form} id={idPrefix + FORM_ID} layout="vertical">
        <Form.Item
          initialValue={sourceWorkspaceId ?? 1}
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
                    <Label truncate={{ tooltip: true }}>{workspace.name}</Label>
                    {workspace.archived && <Icon name="archive" title="Archived" />}
                  </div>
                </Option>
              );
            })}
          </Select>
        </Form.Item>
        {workspaceId && workspaceId !== 1 && (
          <Form.Item
            initialValue={sourceProjectId}
            label="Project"
            name="projectId"
            rules={[{ message: 'Project is required', required: true }]}>
            {Loadable.match(loadableProjects, {
              Failed: () => <div>Failed to load</div>,
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
                        <Label truncate={{ tooltip: true }}>{project.name}</Label>
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

export default FlatRunMoveModalComponent;
