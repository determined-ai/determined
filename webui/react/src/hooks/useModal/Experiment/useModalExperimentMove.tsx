import { Empty, notification, Select, Typography } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import Link from 'components/Link';
import SelectFilter from 'components/SelectFilter';
import useSettings, { BaseType, SettingsConfig } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWorkspaceProjects, getWorkspaces, moveExperiment } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { DetailedUser, Project, Workspace } from 'types';
import handleError from 'utils/error';

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

export const settingsConfig: SettingsConfig = {
  settings: [
    {
      defaultValue: undefined,
      key: 'workspaceId',
      skipUrlEncoding: true,
      storageKey: 'workspaceId',
      type: { baseType: BaseType.Integer },
    },
    {
      defaultValue: undefined,
      key: 'projectId',
      skipUrlEncoding: true,
      storageKey: 'projectId',
      type: { baseType: BaseType.Integer },
    },
  ],
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
  const {
    settings: destSettings,
    updateSettings: updateDestSettings,
  } = useSettings<Settings>(settingsConfig);
  const [ sourceProjectId, setSourceProjectId ] = useState<number|undefined>();
  const [ experimentIds, setExperimentIds ] = useState<number[]>();
  const [ workspaces, setWorkspaces ] = useState<Workspace[]>([]);
  const [ projects, setProjects ] = useState<Project[]>([]);

  const handleClose = useCallback(() => onClose?.(), [ onClose ]);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose: handleClose });

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({ limit: 0 });
      setWorkspaces(response.workspaces);
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to fetch workspaces.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, []);

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
  }, [ destSettings.workspaceId ]);

  useEffect(() => {
    if (modalRef.current) fetchWorkspaces();
  }, [ fetchWorkspaces, modalRef ]);

  const handleWorkspaceSelect = useCallback((workspaceId: SelectValue) => {
    updateDestSettings({
      projectId: (workspaceId === 1 && sourceProjectId !== 1) ? 1 : undefined,
      workspaceId: workspaceId as number,
    });
    setProjects([]);
  }, [ sourceProjectId, updateDestSettings ]);

  const handleProjectSelect = useCallback(
    (project: Project) => {
      if (project.archived || project.id === sourceProjectId) return;
      updateDestSettings({ projectId: project.id });
    },
    [ sourceProjectId, updateDestSettings ],
  );

  useEffect(() => {
    if (modalRef.current) fetchProjects();
  }, [ fetchProjects, modalRef ]);

  const renderRow = useCallback(({ index, style }) => {
    const disabled = projects[index].archived || projects[index].id === sourceProjectId;
    const selected = projects[index].id === destSettings.projectId;
    return (
      <li
        className={disabled ? css.disabled : selected ? css.selected : css.default}
        style={style}
        onClick={() => handleProjectSelect(projects[index])}>
        <Typography.Text
          disabled={disabled}
          ellipsis={true}>
          {projects[index].name}
        </Typography.Text>
        {projects[index].archived && <Icon name="archive" />}
        {projects[index].id === sourceProjectId && <Icon name="checkmark" />}
      </li>
    );
  }, [ destSettings.projectId, handleProjectSelect, projects, sourceProjectId ]);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <div>
          <label className={css.label} htmlFor="workspace">Workspace</label>
          <SelectFilter
            id="workspace"
            placeholder="Select a destination workspace."
            showSearch={false}
            style={{ width: '100%' }}
            value={destSettings.workspaceId}
            onSelect={handleWorkspaceSelect}>
            {workspaces.map((workspace) => {
              return (
                <Option
                  disabled={workspace.archived}
                  key={workspace.id}
                  value={workspace.id}>
                  <div className={workspace.archived ? css.workspaceOptionDisabled : ''}>
                    <Typography.Text
                      ellipsis={true}>
                      {workspace.name}
                    </Typography.Text>
                    {workspace.archived && <Icon name="archive" />}
                  </div>
                </Option>
              );
            })}
          </SelectFilter>
        </div>
        {destSettings.workspaceId !== 1 && (
          <div>
            <label className={css.label} htmlFor="project">Project</label>
            {destSettings.workspaceId === undefined ? (
              <div className={css.emptyContainer}>
                <Empty description="Select a workspace" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              </div>
            ) :
              projects.length === 0 ? (
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
  }, [ handleWorkspaceSelect, projects.length, renderRow, destSettings.workspaceId, workspaces ]);

  const closeNotification = useCallback(() => notification.destroy(), []);

  const handleOk = useCallback(async () => {
    if (!destSettings.projectId || !experimentIds?.length) return;

    const results = await Promise.allSettled(
      experimentIds.map((experimentId) =>
        moveExperimentWithHandler(experimentId, destSettings.projectId as number)),
    );
    const numFailures = results.filter((res) => (
      res.status !== 'fulfilled' || res.value === 1
    )).length;

    const experimentText = experimentIds.length === 1
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
              {numFailures} out of {experimentIds.length} experiments failed to move
              to project {destinationProjectName}
            </p>
            <Link path={paths.projectDetails(destSettings.projectId)}>View Project</Link>
          </div>
        ),
        key: 'move-notification',
        message: 'Partial Move Failure',
      });
    }
  }, [ closeNotification, destSettings.projectId, experimentIds, projects ]);

  const getModalProps = useCallback((experimentIds, destinationProjectId): ModalFuncProps => {
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
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback(
    ({
      initialModalProps,
      experimentIds,
      sourceWorkspaceId,
      sourceProjectId,
    }: ShowModalProps = {}) => {
      setExperimentIds(experimentIds);
      if (!destSettings.workspaceId)
        updateDestSettings({ projectId: undefined, workspaceId: sourceWorkspaceId });
      setSourceProjectId(sourceProjectId);
      fetchWorkspaces();
      fetchProjects();
      openOrUpdate({
        ...getModalProps(experimentIds, destSettings.projectId),
        ...initialModalProps,
      });
    },
    [
      fetchWorkspaces,
      getModalProps,
      openOrUpdate,
      fetchProjects,
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
    if (modalRef.current) openOrUpdate(getModalProps(experimentIds, destSettings.projectId));
  }, [ destSettings.projectId, getModalProps, modalRef, openOrUpdate, experimentIds ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalExperimentMove;
