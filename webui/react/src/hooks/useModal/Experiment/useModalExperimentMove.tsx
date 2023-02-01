import { notification, Select, Typography } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import Empty from 'components/kit/Empty';
import Link from 'components/Link';
import SelectFilter from 'components/SelectFilter';
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
import { DetailedUser, Project } from 'types';
import { Loadable } from 'utils/loadable';

import css from './useModalExperimentMove.module.scss';

const { Option } = Select;

interface Props {
  onClose?: () => void;
  user?: DetailedUser;
}

export interface ShowModalProps {
  experimentIds?: number[];
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
  const [workspaceId, setWorkspaceId] = useState<number>(1);
  const [projectId, setProjectId] = useState<number | null>(null);

  const id = projectId ?? 1;

  const experimentSettingsConfig = useMemo(() => settingsConfigForProject(id), [id]);

  const { settings: projectSettings, updateSettings: updateProjectSettings } =
    useSettings<ExperimentListSettings>(experimentSettingsConfig);
  const [sourceProjectId, setSourceProjectId] = useState<number | undefined>();
  const [experimentIds, setExperimentIds] = useState<number[]>();
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
  }, [workspaceId, fetchWorkspaces]);

  useEffect(() => {
    ensureProjectsFetched(workspaceId);
  }, [workspaceId, ensureProjectsFetched]);

  const handleWorkspaceSelect = useCallback(
    (workspaceId: SelectValue) => {
      setProjectId(workspaceId === 1 && sourceProjectId !== 1 ? 1 : null);
      if (workspaceId !== undefined && typeof workspaceId === 'number') {
        setWorkspaceId(workspaceId);
      }
    },
    [sourceProjectId],
  );

  const handleProjectSelect = useCallback(
    (project: Project) => {
      if (project.archived || project.id === sourceProjectId) return;
      setProjectId(project.id);
    },
    [sourceProjectId],
  );

  const renderRow = useCallback(
    ({ index, style }: { index: number; style: React.CSSProperties }) => {
      return Loadable.match(projects, {
        Loaded: (projects) => {
          const disabled = projects[index].archived || projects[index].id === sourceProjectId;
          const selected = projects[index].id === projectId;
          return (
            <li
              className={disabled ? css.disabled : selected ? css.selected : css.default}
              style={style}
              onClick={() => handleProjectSelect(projects[index])}>
              <Typography.Text disabled={disabled} ellipsis={true}>
                {projects[index].name}
              </Typography.Text>
              {projects[index].archived && <Icon name="archive" />}
              {projects[index].id === sourceProjectId && <Icon name="checkmark" />}
            </li>
          );
        },
        NotLoaded: () => <Spinner spinning />,
      });
    },
    [projectId, handleProjectSelect, projects, sourceProjectId],
  );
  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <div>
          <label className={css.label} htmlFor="workspace">
            Workspace
          </label>
          <SelectFilter
            id="workspace"
            placeholder="Select a destination workspace."
            showSearch={false}
            style={{ width: '100%' }}
            value={workspaceId ?? undefined}
            onSelect={handleWorkspaceSelect}>
            {Loadable.getOrElse([], workspaces).map((workspace) => {
              return (
                <Option disabled={workspace.archived} key={workspace.id} value={workspace.id}>
                  <div className={workspace.archived ? css.workspaceOptionDisabled : ''}>
                    <Typography.Text ellipsis={true}>{workspace.name}</Typography.Text>
                    {workspace.archived && <Icon name="archive" />}
                  </div>
                </Option>
              );
            })}
          </SelectFilter>
        </div>
        {workspaceId && workspaceId !== 1 && (
          <div>
            <label className={css.label} htmlFor="project">
              Project
            </label>
            {workspaceId === undefined ? (
              <div className={css.emptyContainer}>
                <Empty description="Select a workspace" icon="error" />
              </div>
            ) : Loadable.quickMatch(projects, false, (ps) => ps.length === 0) ? (
              <div className={css.emptyContainer}>
                <Empty description="Workspace contains no projects" icon="error" />
              </div>
            ) : (
              <List
                className={css.listContainer}
                height={200}
                innerElementType="ul"
                itemCount={Loadable.quickMatch(projects, 1, (ps) => ps.length)}
                itemSize={24}
                width="100%">
                {renderRow}
              </List>
            )}
          </div>
        )}
      </div>
    );
  }, [handleWorkspaceSelect, projects, renderRow, workspaceId, workspaces]);

  const closeNotification = useCallback(() => notification.destroy(), []);

  const handleOk = useCallback(async () => {
    if (
      !projectId ||
      !experimentIds?.length ||
      !projectSettings.pinned ||
      Loadable.isLoading(projects)
    )
      return;

    const results = await Promise.allSettled(
      experimentIds.map((experimentId) => moveExperimentWithHandler(experimentId, projectId)),
    );
    const numFailures = results.filter(
      (res) => res.status !== 'fulfilled' || res.value === 1,
    ).length;

    const experimentText =
      experimentIds.length === 1
        ? `Experiment ${experimentIds[0]}`
        : `${experimentIds.length} experiments`;

    const destinationProjectName = projects.data.find((p) => p.id === projectId)?.name ?? '';

    if (numFailures === 0) {
      notification.open({
        btn: null,
        description: (
          <div onClick={closeNotification}>
            <p>
              {experimentText} moved to project {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(projectId)}>View Project</Link>
          </div>
        ),
        message: 'Move Success',
      });
      if (sourceProjectId) {
        const newPinned = { ...projectSettings.pinned };
        const pinSet = new Set(newPinned[sourceProjectId]);
        for (const experimentId of experimentIds) {
          pinSet.delete(experimentId);
        }
        newPinned[sourceProjectId] = Array.from(pinSet);
        updateProjectSettings({ pinned: newPinned });
      }
    } else if (numFailures === experimentIds.length) {
      notification.warn({
        description: `Unable to move ${experimentText}`,
        message: 'Move Failure',
      });
    } else {
      notification.warn({
        description: (
          <div onClick={closeNotification}>
            <p>
              {numFailures} out of {experimentIds.length} experiments failed to move to project{' '}
              {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(projectId)}>View Project</Link>
          </div>
        ),
        key: 'move-notification',
        message: 'Partial Move Failure',
      });
    }
  }, [
    closeNotification,
    experimentIds,
    projectId,
    projectSettings.pinned,
    projects,
    sourceProjectId,
    updateProjectSettings,
  ]);

  const getModalProps = useCallback(
    (experimentIds?: number[], destinationProjectId?: number): ModalFuncProps => {
      const pluralizer = experimentIds?.length && experimentIds?.length > 1 ? 's' : '';
      return {
        closable: true,
        content: modalContent,
        icon: null,
        okButtonProps: { disabled: !destinationProjectId },
        okText: `Move Experiment${pluralizer}`,
        onOk: handleOk,
        title: `Move Experiment${pluralizer}`,
      };
    },
    [handleOk, modalContent],
  );

  const modalOpen = useCallback(
    ({
      initialModalProps,
      experimentIds,
      sourceWorkspaceId,
      sourceProjectId,
    }: ShowModalProps = {}) => {
      if (!workspaceId || projectId) return;

      setExperimentIds(experimentIds);
      if (!workspaceId) setProjectId(null);
      setWorkspaceId(sourceWorkspaceId ?? 1);

      setSourceProjectId(sourceProjectId);

      openOrUpdate({
        ...getModalProps(experimentIds, projectId ?? undefined),
        ...initialModalProps,
      });
    },
    [getModalProps, openOrUpdate, projectId, workspaceId],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(experimentIds, projectId ?? undefined));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, projects._tag, workspaceId, experimentIds]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalExperimentMove;
