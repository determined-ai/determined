import { Dropdown } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon';
import { ValueOf } from 'shared/types';

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
  const handleDownloadClick = useCallback(() => onDownload?.(), [onDownload]);

  const handleDeleteClick = useCallback(() => onDelete?.(), [onDelete]);

  const ModelVersionActionMenu: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      DeleteVersion: 'delete-version',
      Download: 'download',
    } as const;

    const funcs = {
      [MenuKey.Download]: () => {
        handleDownloadClick();
      },
      [MenuKey.DeleteVersion]: () => {
        handleDeleteClick();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const menuItems: MenuProps['items'] = [
      { key: MenuKey.Download, label: 'Download' },
      { danger: true, key: MenuKey.DeleteVersion, label: 'Deregister Version' },
    ];

    return { items: menuItems, onClick: onItemClick };
  }, [handleDeleteClick, handleDownloadClick]);

  return children ? (
    <Dropdown
      menu={ModelVersionActionMenu}
      placement="bottomLeft"
      trigger={trigger ?? ['contextMenu', 'click']}
      onVisibleChange={onVisibleChange}>
      {children}
    </Dropdown>
  ) : (
    <div
      className={[css.base, className].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown
        menu={ModelVersionActionMenu}
        placement="bottomRight"
        trigger={trigger ?? ['click']}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
    </div>
  );
};

export default ModelVersionActionDropdown;
