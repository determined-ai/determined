import { Dropdown, Menu } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo } from 'react';

import useModalModelDownload from 'hooks/useModal/Model/useModalModelDownload';
import useModalModelVersionDelete from 'hooks/useModal/Model/useModalModelVersionDelete';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon';
import { ModelVersion } from 'types';

interface Props {
  className?: string;
  direction?: 'vertical' | 'horizontal';
  modelVersion: ModelVersion;
  onComplete?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ModelVersionActionDropdown: React.FC<Props> = ({
  modelVersion, children, onVisibleChange,
  className, direction = 'vertical', onComplete, trigger,
}: PropsWithChildren<Props>) => {
  const {
    contextHolder: modalModelDownloadContextHolder,
    modalOpen: openModelDownload,
  } = useModalModelDownload({ modelVersion, onClose: onComplete });

  const {
    contextHolder: modalModelVersionDeleteContextHolder,
    modalOpen: openModelVersionDelete,
  } = useModalModelVersionDelete();

  const handleDownloadModel = useCallback(() => {
    openModelDownload({});
  }, [ openModelDownload ]);

  const handleDeleteClick = useCallback(() => {
    openModelVersionDelete(modelVersion);
  }, [ modelVersion, openModelVersionDelete ]);

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
    <>
      <Dropdown
        overlay={ModelVersionActionMenu}
        placement="bottomLeft"
        trigger={trigger ?? [ 'contextMenu', 'click' ]}
        onVisibleChange={onVisibleChange}>
        {children}
      </Dropdown>
      {modalModelDownloadContextHolder}
      {modalModelVersionDeleteContextHolder}
    </>
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
      {modalModelDownloadContextHolder}
      {modalModelVersionDeleteContextHolder}
    </div>
  );
};

export default ModelVersionActionDropdown;
