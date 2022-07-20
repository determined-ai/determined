import { Dropdown, Menu } from 'antd';
import type { MenuProps } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo } from 'react';

import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon';

interface Props {
  className?: string;
  direction?: 'vertical' | 'horizontal';
  onDelete?: () => void;
  onDownload?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ModelVersionActionDropdown: React.FC<Props> = ({
  children,
  className,
  direction = 'vertical',
  onDelete,
  onDownload,
  onVisibleChange,
  trigger,
}: PropsWithChildren<Props>) => {
  const handleDownloadClick = useCallback(() => onDownload?.(), [ onDownload ]);

  const handleDeleteClick = useCallback(() => onDelete?.(), [ onDelete ]);

  const ModelVersionActionMenu = useMemo(() => {
    const DOWNLOAD = 'download';
    const DELETE_VERSION = 'delete-version';
    const onItemClick: MenuProps['onClick'] = (e) => {
      switch(e.key) {
        case DOWNLOAD:
          handleDownloadClick();
          break;
        case DELETE_VERSION:
          handleDeleteClick();
          break;
        default:
          return;
      }
    };

    return (
      <Menu
        items={[
          { key: DOWNLOAD, label: 'Download' },
          { danger: true, key: DELETE_VERSION, label: 'Deregister Version' },
        ]}
        onClick={onItemClick}
      />
    );
  }, [ handleDeleteClick, handleDownloadClick ]);

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
