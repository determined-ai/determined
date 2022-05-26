import { Dropdown, Menu, Modal } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo } from 'react';

import showModalItemCannotDelete from 'components/ModalItemDelete';
import useModalDownloadModel from 'hooks/useModal/useModalDownloadModel';
import { deleteModelVersion } from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser, ModelItem, ModelVersion } from 'types';
import handleError from 'utils/error';

interface Props {
  className?: string;
  curUser?: DetailedUser;
  direction?: 'vertical' | 'horizontal';
  model?: ModelItem;
  modelVersion: ModelVersion;
  onComplete?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ModelVersionActionDropdown: React.FC<Props> = (
  {
    model, modelVersion, children, curUser, onVisibleChange,
    className, direction = 'vertical', onComplete, trigger,
  }
  : PropsWithChildren<Props>,
) => {
  const { modalOpen: openModelDownload } = useModalDownloadModel({
    modelVersion,
    onClose: onComplete,
  });

  const isDeletable = useMemo(() => {
    return curUser?.isAdmin
    || curUser?.id === model?.userId
    || curUser?.id === modelVersion.userId;
  }, [ curUser?.id, curUser?.isAdmin, model?.userId, modelVersion.userId ]);

  const deleteVersion = useCallback(async (version: ModelVersion) => {
    try {
      await deleteModelVersion({ modelName: version.model.name, versionId: version.id });
    } catch (e) {
      handleError(e, {
        publicSubject: `Unable to delete model version ${version.id}.`,
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, []);

  const showConfirmDelete = useCallback((version: ModelVersion) => {
    Modal.confirm({
      closable: true,
      content: `Are you sure you want to delete this version "Version ${version.version}"
      from this model?`,
      icon: null,
      maskClosable: true,
      okText: 'Delete Version',
      okType: 'danger',
      onOk: () => deleteVersion(version),
      title: 'Confirm Delete',
    });
  }, [ deleteVersion ]);

  const handleDownloadModel = useCallback(() => {
    openModelDownload({});
  }, [ openModelDownload ]);

  const handleDeleteClick = useCallback(() => {
    isDeletable ? showConfirmDelete(modelVersion) : showModalItemCannotDelete();
  }, [ isDeletable, modelVersion, showConfirmDelete ]);

  const ModelVersionActionMenu = useMemo(() => {
    return (
      <Menu>
        <Menu.Item
          key="download"
          onClick={handleDownloadModel}>
          Download
        </Menu.Item>
        <Menu.Item
          danger
          key="delete-version"
          onClick={handleDeleteClick}>
          Deregister Version
        </Menu.Item>
      </Menu>
    );
  }, [ handleDownloadModel, handleDeleteClick ]);

  return children ? (
    <Dropdown
      overlay={ModelVersionActionMenu}
      placement="bottomLeft"
      trigger={trigger ?? [ 'contextMenu', 'click' ]}
      onVisibleChange={onVisibleChange}>
      {children}
    </Dropdown>
  ) : (
    <div
      className={[ css.base, className ].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown
        overlay={ModelVersionActionMenu}
        placement="bottomRight"
        trigger={trigger ?? [ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
    </div>
  );
};

export default ModelVersionActionDropdown;
