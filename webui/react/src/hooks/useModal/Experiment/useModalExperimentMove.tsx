import { Typography } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Form from 'components/kit/Form';
import Select, { Option } from 'components/kit/Select';
import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import { ExperimentListSettings, settingsConfigForProject } from 'pages/ExperimentList.settings';
import { paths } from 'routes/utils';
import { moveExperiment } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { useEnsureWorkspaceProjectsFetched, useWorkspaceProjects } from 'stores/projects';
import { useEnsureWorkspacesFetched, useWorkspaces } from 'stores/workspaces';
import { DetailedUser } from 'types';
import { notification } from 'utils/dialogApi';
import { Loadable } from 'utils/loadable';

import css from './useModalExperimentMove.module.scss';

type FormInputs = {
  projectId?: number;
  workspaceId: number;
};

interface Props {
  onClose?: () => void;
  user?: DetailedUser;
}

export interface ShowModalProps {
  experimentIds: number[];
  initialModalProps?: ModalFuncProps;
  sourceProjectId?: number;
  sourceWorkspaceId?: number;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

const moveExperimentWithHandler = async (
  experimentId: number,
  destinationProjectId: number,
): Promise<number> => {
  try {
    await moveExperiment({ destinationProjectId, experimentId });
    return 0;
  } catch (e) {
    return 1;
  }
};

const useModalExperimentMove = ({ onClose }: Props): ModalHooks => {
  const canceler = useRef(new AbortController());
  const [form] = Form.useForm<FormInputs>();
  const workspaceId = Form.useWatch('workspaceId', form);
  const projectId = Form.useWatch('projectId', form);

  const id = projectId ?? 1;

  const experimentSettingsConfig = useMemo(() => settingsConfigForProject(id), [id]);

  const { settings: projectSettings, updateSettings: updateProjectSettings } =
    useSettings<ExperimentListSettings>(experimentSettingsConfig);
  const [experimentIds, setExperimentIds] = useState<number[]>([]);
  const { canMoveExperimentsTo } = usePermissions();
  const loadableWorkspaces = useWorkspaces({ archived: false });
  const workspaces = Loadable.map(loadableWorkspaces, (ws) =>
    ws.filter((w) => canMoveExperimentsTo({ destination: { id: w.id } })),
  );
  const projects = useWorkspaceProjects(workspaceId);
  const ensureProjectsFetched = useEnsureWorkspaceProjectsFetched(canceler.current);
  const fetchWorkspaces = useEnsureWorkspacesFetched(canceler.current);

  const handleClose = useCallback(() => onClose?.(), [onClose]);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose: handleClose });

  useEffect(() => {
    fetchWorkspaces();
  }, [fetchWorkspaces]);

  useEffect(() => {
    ensureProjectsFetched(workspaceId);
  }, [workspaceId, ensureProjectsFetched]);

  const modalContent = useMemo(() => {
    return (
      <Form className={css.base} form={form} layout="vertical">
        <Form.Item
          label="Workspace"
          name="workspaceId"
          rules={[{ message: 'Workspace is required', required: true }]}>
          <Select
            id="workspace"
            placeholder="Select a destination workspace."
            onChange={() => form.resetFields(['projectId'])}>
            {Loadable.getOrElse([], workspaces).map((workspace) => {
              return (
                <Option disabled={workspace.archived} key={workspace.id} value={workspace.id}>
                  <div className={workspace.archived ? css.optionDisabled : undefined}>
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
            {Loadable.match(projects, {
              Loaded: (projects) => (
                <Select placeholder="Select a destination project.">
                  {projects.map((project) => (
                    <Option disabled={project.archived} key={project.id} value={project.id}>
                      <div className={project.archived ? css.optionDisabled : undefined}>
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
    );
  }, [form, projects, workspaceId, workspaces]);

  const closeNotification = useCallback(() => notification.destroy(), []);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();
    const projId = values.projectId ?? 1;

    const results = await Promise.allSettled(
      experimentIds.map((experimentId) => moveExperimentWithHandler(experimentId, projId)),
    );
    const numFailures = results.filter(
      (res) => res.status !== 'fulfilled' || res.value === 1,
    ).length;

    const experimentText =
      experimentIds.length === 1
        ? `Experiment ${experimentIds[0]}`
        : `${experimentIds.length} experiments`;

    const destinationProjectName =
      Loadable.getOrElse([], projects).find((p) => p.id === projId)?.name ?? '';

    if (numFailures === 0) {
      notification.open({
        btn: null,
        description: (
          <div onClick={closeNotification}>
            <p>
              {experimentText} moved to project {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(projId)}>View Project</Link>
          </div>
        ),
        message: 'Move Success',
      });
      if (projId) {
        const newPinned = { ...projectSettings.pinned };
        const pinSet = new Set(newPinned?.[projId] ?? []);
        for (const experimentId of experimentIds) {
          pinSet.delete(experimentId);
        }
        newPinned[projId] = Array.from(pinSet);
        updateProjectSettings({ pinned: newPinned });
      }
    } else if (numFailures === experimentIds.length) {
      notification.warning({
        description: `Unable to move ${experimentText}`,
        message: 'Move Failure',
      });
    } else {
      notification.warning({
        description: (
          <div onClick={closeNotification}>
            <p>
              {numFailures} out of {experimentIds.length} experiments failed to move to project{' '}
              {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(projId)}>View Project</Link>
          </div>
        ),
        key: 'move-notification',
        message: 'Partial Move Failure',
      });
    }
    form.resetFields();
  }, [
    closeNotification,
    experimentIds,
    form,
    projectSettings.pinned,
    projects,
    updateProjectSettings,
  ]);

  const getModalProps = useCallback(
    (experimentIds: number[]): ModalFuncProps => {
      const pluralizer = experimentIds.length > 1 ? 's' : '';
      return {
        closable: true,
        content: modalContent,
        icon: null,
        okText: `Move Experiment${pluralizer}`,
        onOk: handleOk,
        title: `Move Experiment${pluralizer}`,
      };
    },
    [handleOk, modalContent],
  );

  const modalOpen = useCallback(
    (
      { initialModalProps, experimentIds, sourceWorkspaceId, sourceProjectId }: ShowModalProps = {
        experimentIds: [],
      },
    ) => {
      setExperimentIds(experimentIds);
      form.setFieldValue('projectId', sourceProjectId);
      form.setFieldValue('workspaceId', sourceWorkspaceId ?? 1);

      openOrUpdate({
        ...getModalProps(experimentIds),
        ...initialModalProps,
      });
    },
    [form, getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(experimentIds));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, projects._tag, workspaceId, experimentIds]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalExperimentMove;
