import { Dropdown, Menu } from 'antd';
import type { MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon';

interface Props {
  children?: React.ReactNode;
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
}: Props) => {
  const handleDownloadClick = useCallback(() => onDownload?.(), [ onDownload ]);

  const handleDeleteClick = useCallback(() => onDelete?.(), [ onDelete ]);

  const ModelVersionActionMenu = useMemo(() => {
    enum MenuKey {
      DOWNLOAD = 'download',
      DELETE_VERSION = 'delete-version'
    }

    const funcs = {
      [MenuKey.DOWNLOAD]: () => { handleDownloadClick(); },
      [MenuKey.DELETE_VERSION]: () => { handleDeleteClick(); },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as MenuKey]();
    };

    const menuItems: MenuProps['items'] = [
      { key: MenuKey.DOWNLOAD, label: 'Download' },
      { danger: true, key: MenuKey.DELETE_VERSION, label: 'Deregister Version' },
    ];

    return <Menu items={menuItems} onClick={onItemClick} />;
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
