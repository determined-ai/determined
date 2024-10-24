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
import React, { useEffect, useId, useState } from 'react';

import Link from 'components/Link';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { moveSearches } from 'services/api';
import { V1MoveSearchesRequest } from 'services/api-ts-sdk';
import projectStore from 'stores/projects';
import workspaceStore from 'stores/workspaces';
import { Project, SelectionType, XOR } from 'types';
import handleError from 'utils/error';
import { getIdsFilter as getExperimentIdsFilter } from 'utils/experiment';
import { capitalize, pluralizer } from 'utils/string';

import { INIT_FORMSET } from './FilterForm/components/FilterFormStore';
import { FilterFormSet } from './FilterForm/components/type';

const FORM_ID = 'move-experiment-form';

type FormInputs = {
  projectId?: number;
  workspaceId?: number;
};

interface BaseProps {
  onSubmit?: (successfulIds?: number[]) => void;
  selectionSize: number;
  sourceProjectId: number;
  sourceWorkspaceId?: number;
}

type Props = BaseProps &
  XOR<{ experimentIds: number[] }, { selection: SelectionType; tableFilters: string }>;

const ExperimentMoveModalComponent: React.FC<Props> = ({
  experimentIds,
  selection,
  selectionSize,
  tableFilters,
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
  const f_flat_runs = useFeature().isOn('flat_runs');

  useEffect(() => {
    setDisabled(workspaceId !== 1 && !projectId);
  }, [workspaceId, projectId, sourceProjectId, sourceWorkspaceId]);

  const { canMoveExperimentsTo } = usePermissions();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.unarchived)).filter((w) =>
    canMoveExperimentsTo({ destination: { id: w.id } }),
  );
  const loadableProjects: Loadable<List<Project>> = useObservable(
    projectStore.getProjectsByWorkspace(workspaceId),
  );

  useEffect(() => workspaceStore.fetch(), []);

  useEffect(() => {
    if (workspaceId !== undefined) {
      projectStore.fetch(workspaceId, undefined, true);
    }
  }, [workspaceId]);

  // use plurals for indeterminate case
  const pluralizerArgs = f_flat_runs
    ? (['search', 'searches'] as const)
    : (['experiment'] as const);
  // we use apply instead of a direct call here because typescript errors when you spread a tuple into arguments
  const plural = pluralizer.apply(null, [selectionSize, ...pluralizerArgs]);
  const actionCopy = `Move ${capitalize(plural)}`;

  const handleSubmit = async () => {
    if (workspaceId === sourceWorkspaceId && projectId === sourceProjectId) {
      openToast({ title: 'No changes to save.' });
      return;
    }
    const values = await form.validateFields();
    const projId = values.projectId ?? 1;

    const moveSearchesArgs: V1MoveSearchesRequest = {
      destinationProjectId: projId,
      sourceProjectId,
    };

    if (tableFilters !== undefined) {
      const filterFormSet =
        selection.type === 'ALL_EXCEPT'
          ? (JSON.parse(tableFilters) as FilterFormSet)
          : INIT_FORMSET;
      const filter = getExperimentIdsFilter(filterFormSet, selection);
      moveSearchesArgs.filter = JSON.stringify(filter);
    } else {
      moveSearchesArgs.searchIds = experimentIds;
    }

    const results = await moveSearches(moveSearchesArgs);

    onSubmit?.(results.successful);

    const numSuccesses = results.successful.length;
    const numFailures = results.failed.length;

    const destinationProjectName =
      Loadable.getOrElse(List<Project>(), loadableProjects).find((p) => p.id === projId)?.name ??
      '';

    if (numSuccesses === 0 && numFailures === 0) {
      openToast({
        description: `No selected ${plural} were eligible for moving`,
        title: `No eligible ${plural}`,
      });
    } else if (numFailures === 0) {
      openToast({
        closeable: true,
        description: `${results.successful.length} ${pluralizer.apply(null, [results.successful.length, ...pluralizerArgs])} moved to project ${destinationProjectName}`,
        link: <Link path={paths.projectDetails(projId)}>View Project</Link>,
        title: 'Move Success',
      });
    } else if (numSuccesses === 0) {
      openToast({
        description: `Unable to move ${numFailures} ${pluralizer.apply(null, [numFailures, ...pluralizerArgs])}`,
        severity: 'Warning',
        title: 'Move Failure',
      });
    } else {
      openToast({
        closeable: true,
        description: `${numFailures} out of ${
          numFailures + numSuccesses
        } eligible ${plural} failed to move
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
        text: actionCopy,
      }}
      title={actionCopy}>
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

export default ExperimentMoveModalComponent;
