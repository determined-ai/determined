import { Dropdown, Menu } from 'antd';
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
    return (
      <Menu>
        <Menu.Item
          key="download"
          onClick={handleDownloadClick}>
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
