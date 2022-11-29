import { Empty, notification, Select, Typography } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { SelectValue } from 'antd/lib/select';
import { number, undefined as undefinedType, union } from 'io-ts';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import Link from 'components/Link';
import SelectFilter from 'components/SelectFilter';
import usePermissions from 'hooks/usePermissions';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import projectDetailConfigSettings, {
  ProjectDetailsSettings,
} from 'pages/OldProjectDetails.settings';
import { paths } from 'routes/utils';
import { getWorkspaceProjects, moveExperiment } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { useEnsureWorkspacesFetched, useWorkspaces } from 'stores/workspaces';
import { DetailedUser, Project } from 'types';
import handleError from 'utils/error';
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

export interface Settings {
  projectId?: number;
  workspaceId?: number;
}

export const settingsConfig: SettingsConfig<Settings> = {
  applicableRoutespace: 'experiment-destination',
  settings: {
    projectId: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'projectId',
      type: union([undefinedType, number]),
    },
    workspaceId: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'workspaceId',
      type: union([undefinedType, number]),
    },
  },
  storagePath: 'experiment-destination',
};

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
  const { settings: destSettings, updateSettings: updateDestSettings } =
    useSettings<Settings>(settingsConfig);

  const { settings: projectSettings, updateSettings: updateProjectSettings } =
    useSettings<ProjectDetailsSettings>(projectDetailConfigSettings);
  const [sourceProjectId, setSourceProjectId] = useState<number | undefined>();
  const [experimentIds, setExperimentIds] = useState<number[]>();
  const [projects, setProjects] = useState<Project[]>([]);
  const { canMoveExperimentsTo } = usePermissions();
  const workspaces = Loadable.map(useWorkspaces(), (ws) =>
    ws.filter((w) => !w.immutable && canMoveExperimentsTo({ destination: { id: w.id } })),
  );
  const ensureWorkspacesFetched = useEnsureWorkspacesFetched(canceler.current);

  const handleClose = useCallback(() => onClose?.(), [onClose]);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose: handleClose });

  const fetchProjects = useCallback(async () => {
    if (!destSettings.workspaceId) return;
    try {
      const response = await getWorkspaceProjects({
        id: destSettings.workspaceId,
        limit: 0,
        // users: (!user || user.isAdmin) ? [] : [ user.username ],
      });
      setProjects((prev) => (isEqual(prev, response.projects) ? prev : response.projects));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to fetch projects.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [destSettings.workspaceId]);

  useEffect(() => {
    ensureWorkspacesFetched();
    fetchProjects();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleWorkspaceSelect = useCallback(
    (workspaceId: SelectValue) => {
      updateDestSettings({
        projectId: workspaceId === 1 && sourceProjectId !== 1 ? 1 : undefined,
        workspaceId: workspaceId as number,
      });
      setProjects([]);
    },
    [sourceProjectId, updateDestSettings],
  );

  const handleProjectSelect = useCallback(
    (project: Project) => {
      if (project.archived || project.id === sourceProjectId) return;
      updateDestSettings({ projectId: project.id });
    },
    [sourceProjectId, updateDestSettings],
  );

  const renderRow = useCallback(
    ({ index, style }: { index: number; style: React.CSSProperties }) => {
      if (!destSettings.projectId) return <Spinner spinning />;

      const disabled = projects[index].archived || projects[index].id === sourceProjectId;
      const selected = projects[index].id === destSettings.projectId;
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
    [destSettings.projectId, handleProjectSelect, projects, sourceProjectId],
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
            value={destSettings.workspaceId}
            onSelect={handleWorkspaceSelect}>
            {Loadable.getOrElse([], workspaces).map((workspace) => {
              // TODO loading state
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
        {destSettings.workspaceId && destSettings.workspaceId !== 1 && (
          <div>
            <label className={css.label} htmlFor="project">
              Project
            </label>
            {destSettings.workspaceId === undefined ? (
              <div className={css.emptyContainer}>
                <Empty description="Select a workspace" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              </div>
            ) : projects.length === 0 ? (
              <div className={css.emptyContainer}>
                <Empty
                  description="Workspace contains no projects"
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              </div>
            ) : (
              <List
                className={css.listContainer}
                height={200}
                innerElementType="ul"
                itemCount={projects.length}
                itemSize={24}
                width="100%">
                {renderRow}
              </List>
            )}
          </div>
        )}
      </div>
    );
  }, [handleWorkspaceSelect, projects.length, renderRow, destSettings.workspaceId, workspaces]);

  const closeNotification = useCallback(() => notification.destroy(), []);

  const handleOk = useCallback(async () => {
    if (!destSettings.projectId || !experimentIds?.length || !projectSettings.pinned) return;

    const results = await Promise.allSettled(
      experimentIds.map((experimentId) =>
        moveExperimentWithHandler(experimentId, destSettings.projectId as number),
      ),
    );
    const numFailures = results.filter(
      (res) => res.status !== 'fulfilled' || res.value === 1,
    ).length;

    const experimentText =
      experimentIds.length === 1
        ? `Experiment ${experimentIds[0]}`
        : `${experimentIds.length} experiments`;

    const destinationProjectName =
      projects.find((p) => p.id === destSettings.projectId)?.name ?? '';

    if (numFailures === 0) {
      notification.open({
        btn: null,
        description: (
          <div onClick={closeNotification}>
            <p>
              {experimentText} moved to project {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(destSettings.projectId)}>View Project</Link>
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
            <Link path={paths.projectDetails(destSettings.projectId)}>View Project</Link>
          </div>
        ),
        key: 'move-notification',
        message: 'Partial Move Failure',
      });
    }
  }, [
    closeNotification,
    destSettings.projectId,
    experimentIds,
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
      if (!destSettings.workspaceId || destSettings.projectId) return;

      setExperimentIds(experimentIds);
      if (!destSettings.workspaceId)
        updateDestSettings({ projectId: undefined, workspaceId: sourceWorkspaceId });
      setSourceProjectId(sourceProjectId);
      if (!projects.length) fetchProjects();
      openOrUpdate({
        ...getModalProps(experimentIds, destSettings.projectId),
        ...initialModalProps,
      });
    },
    [
      getModalProps,
      openOrUpdate,
      fetchProjects,
      projects.length,
      destSettings.projectId,
      destSettings.workspaceId,
      updateDestSettings,
    ],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current && destSettings.projectId)
      openOrUpdate(getModalProps(experimentIds, destSettings.projectId));
  }, [destSettings.projectId, getModalProps, modalRef, openOrUpdate, experimentIds]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalExperimentMove;
