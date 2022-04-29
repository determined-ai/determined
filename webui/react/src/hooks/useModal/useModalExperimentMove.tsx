import { Empty, Select, Typography } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import { SelectValue } from 'antd/lib/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { FixedSizeList as List } from 'react-window';

import SelectFilter from 'components/SelectFilter';
import useModal, { ModalHooks } from 'hooks/useModal/useModal';
import { getWorkspaceProjects, getWorkspaces, moveExperiment } from 'services/api';
import { ExperimentTask, Project, Workspace } from 'types';
import { isEqual } from 'utils/data';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import css from './useModalExperimentMove.module.scss';

const { Option } = Select;

interface Props {
  experiment?: ExperimentTask;
  onClose?: () => void;
}

const useModalExperimentMove = ({ onClose, experiment }: Props): ModalHooks => {
  const [ selectedWorkspaceId, setSelectedWorkspaceId ] = useState<number>();
  const [ destinationProjectId, setDestinationProjectId ] = useState<number>();
  const [ workspaces, setWorkspaces ] = useState<Workspace[]>([]);
  const [ projects, setProjects ] = useState<Project[]>([]);

  const handleClose = useCallback(() => {
    onClose?.();
  }, [ onClose ]);

  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose: handleClose });

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({ archived: false, limit: 0 });
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
    if (!selectedWorkspaceId) return;
    try {
      const response = await getWorkspaceProjects({
        archived: false,
        id: selectedWorkspaceId,
        limit: 0,
      });
      setProjects(prev => {
        const withoutCurrent = response.projects.filter(p =>
          p.id !== experiment?.projectId);
        if (isEqual(prev, withoutCurrent)) return prev;
        return withoutCurrent;
      });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to fetch projects.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ experiment?.projectId, selectedWorkspaceId ]);

  useEffect(() => {
    if (modalRef.current) fetchWorkspaces();
  }, [ fetchWorkspaces, modalRef ]);

  const handleWorkspaceSelect = useCallback((value: SelectValue) => {
    setSelectedWorkspaceId(value as number);
    setProjects([]);
  }, []);

  const handleProjectSelect = useCallback((value: number) => {
    setDestinationProjectId(value);
  }, []);

  useEffect(() => {
    if (modalRef.current) fetchProjects();
  }, [ fetchProjects, modalRef ]);

  const renderRow = useCallback(({ index, style }) => {
    return (
      <li
        style={{
          ...style,
          backgroundColor: projects[index].id === destinationProjectId ?
            'var(--theme-colors-monochrome-16)' :
            undefined,
        }}
        onClick={() => handleProjectSelect(projects[index].id)}>
        <Typography.Text ellipsis={true}>{projects[index].name}</Typography.Text>
      </li>
    );
  }, [ destinationProjectId, handleProjectSelect, projects ]);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <div>
          <label className={css.label} htmlFor="workspace">Workspace</label>
          <SelectFilter
            id="workspace"
            placeholder="Select a destination workspace."
            style={{ width: '100%' }}
            onChange={handleWorkspaceSelect}>
            {workspaces.map(workspace => {
              return (
                <Option key={workspace.id} value={workspace.id}>
                  {workspace.name}
                </Option>
              );
            })}
          </SelectFilter>
        </div>
        <div>
          <label className={css.label} htmlFor="project">Project</label>
          {selectedWorkspaceId === undefined ? (
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
      </div>
    );
  }, [ handleWorkspaceSelect, projects.length, renderRow, selectedWorkspaceId, workspaces ]);

  const handleOk = useCallback(async () => {
    if (!destinationProjectId) return;
    try {
      await moveExperiment({ destinationProjectId, experimentId: parseInt(experiment?.id ?? '0') });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to move experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ destinationProjectId, experiment?.id ]);

  const getModalProps = useCallback((destinationProjectId?: number): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !destinationProjectId },
      okText: 'Move Experiment',
      onOk: handleOk,
      title: 'Move Experiment',
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setSelectedWorkspaceId(undefined);
    setDestinationProjectId(undefined);
    setProjects([]);
    fetchWorkspaces();
    openOrUpdate({ ...getModalProps(undefined), ...initialModalProps });
  }, [ fetchWorkspaces, getModalProps, openOrUpdate ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(destinationProjectId));
  }, [ destinationProjectId, getModalProps, modalRef, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalExperimentMove;
