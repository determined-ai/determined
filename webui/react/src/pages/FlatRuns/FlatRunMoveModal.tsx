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
import React, { Ref, useCallback, useEffect, useId, useRef } from 'react';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { FilterFormSet } from 'components/FilterForm/components/type';
import RunFilterInterstitialModalComponent, {
  ControlledModalRef,
} from 'components/RunFilterInterstitialModalComponent';
import RunMoveWarningModalComponent, {
  RunMoveWarningFlowRef,
} from 'components/RunMoveWarningModalComponent';
import usePermissions from 'hooks/usePermissions';
import { formStore } from 'pages/FlatRuns/FlatRuns';
import { moveRuns } from 'services/api';
import { V1MoveRunsRequest } from 'services/api-ts-sdk';
import projectStore from 'stores/projects';
import workspaceStore from 'stores/workspaces';
import { BulkActionResult, Project, SelectionType, XOR } from 'types';
import handleError from 'utils/error';
import { getIdsFilter as getRunIdsFilter } from 'utils/flatRun';
import { pluralizer } from 'utils/string';

const FORM_ID = 'move-flat-run-form';

type FormInputs = {
  destinationProjectId?: number;
  destinationWorkspaceId?: number;
};

interface BaseProps {
  selectionSize: number;
  sourceProjectId: number;
  sourceWorkspaceId?: number;
  onSubmit?: (results: BulkActionResult, destinationProjectId: number) => void | Promise<void>;
}

type Props = BaseProps &
  XOR<{ runIds: number[] }, { selection: SelectionType; tableFilters: string }>;

const FlatRunMoveModalComponent: React.FC<Props> = ({
  runIds,
  tableFilters,
  selection,
  selectionSize,
  sourceProjectId,
  sourceWorkspaceId,
  onSubmit,
}: Props) => {
  const controlledModalRef: Ref<ControlledModalRef> = useRef(null);
  const runMoveWarningFlowRef: Ref<RunMoveWarningFlowRef> = useRef(null);
  const idPrefix = useId();
  const { openToast } = useToast();
  const filterFormSetWithoutId = useObservable(formStore.filterFormSetWithoutId);
  const [form] = Form.useForm<FormInputs>();
  const destinationWorkspaceId = Form.useWatch('destinationWorkspaceId', form);
  const destinationProjectId = Form.useWatch('destinationProjectId', form);

  const { canMoveExperimentsTo } = usePermissions();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.unarchived)).filter((w) =>
    canMoveExperimentsTo({ destination: { id: w.id } }),
  );
  const loadableProjects: Loadable<List<Project>> = useObservable(
    projectStore.getProjectsByWorkspace(destinationWorkspaceId),
  );

  useEffect(() => {
    if (destinationWorkspaceId !== undefined) {
      projectStore.fetch(destinationWorkspaceId, undefined, true);
    }
  }, [destinationWorkspaceId]);

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();
    const projId = values.destinationProjectId ?? 1;

    if (destinationWorkspaceId === sourceWorkspaceId && projId === sourceProjectId) {
      openToast({ title: 'No changes to save.' });
      return;
    }

    try {
      const closeReason = (await controlledModalRef.current?.open()) ?? 'failed';
      switch (closeReason) {
        case 'has_search_runs': {
          const closeWarningReason = await runMoveWarningFlowRef.current?.open();
          if (closeWarningReason === 'cancel') {
            openToast({ title: 'Cancelled Move Action' });
            return;
          }
          break;
        }
        case 'no_search_runs':
          break;
        case 'manual':
        case 'failed':
        case 'close':
          openToast({ title: 'Cancelled Move Action' });
          return;
      }

      const moveRunsArgs: V1MoveRunsRequest = {
        destinationProjectId: projId,
        sourceProjectId,
      };

      if (tableFilters !== undefined) {
        const filterFormSet =
          selection.type === 'ALL_EXCEPT'
            ? (JSON.parse(tableFilters) as FilterFormSet)
            : INIT_FORMSET;
        const filter = getRunIdsFilter(filterFormSet, selection);
        moveRunsArgs.filter = JSON.stringify(filter);
      } else {
        moveRunsArgs.runIds = runIds;
      }

      const results = await moveRuns(moveRunsArgs);
      await onSubmit?.(results, projId);
      form.resetFields();
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to move runs' });
    }
  }, [
    form,
    destinationWorkspaceId,
    sourceWorkspaceId,
    sourceProjectId,
    openToast,
    tableFilters,
    onSubmit,
    selection,
    runIds,
  ]);

  return (
    <>
      <Modal
        cancel
        size="small"
        submit={{
          disabled: destinationWorkspaceId !== 1 && !destinationProjectId,
          form: idPrefix + FORM_ID,
          handleError,
          handler: handleSubmit,
          text: `Move ${pluralizer(selectionSize, 'Run')}`,
        }}
        title={`Move ${pluralizer(selectionSize, 'Run')}`}>
        <Form form={form} id={idPrefix + FORM_ID} layout="vertical">
          <Form.Item
            initialValue={sourceWorkspaceId ?? 1}
            label="Workspace"
            name="destinationWorkspaceId"
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
          {destinationWorkspaceId !== undefined && destinationWorkspaceId !== 1 && (
            <Form.Item
              label="Project"
              name="destinationProjectId"
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
      <RunMoveWarningModalComponent ref={runMoveWarningFlowRef} />
      <RunFilterInterstitialModalComponent
        filterFormSet={filterFormSetWithoutId}
        projectId={sourceProjectId}
        ref={controlledModalRef}
        selection={selection ?? { selections: runIds, type: 'ONLY_IN' }}
      />
    </>
  );
};

export default FlatRunMoveModalComponent;
