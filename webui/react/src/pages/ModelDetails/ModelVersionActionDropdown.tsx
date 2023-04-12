import { Dropdown } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import usePermissions from 'hooks/usePermissions';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import { ValueOf } from 'shared/types';
import { ModelVersion } from 'types';

interface Props {
  children?: React.ReactNode;
  className?: string;
  direction?: 'vertical' | 'horizontal';
  onDelete?: () => void;
  onDownload?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
  version: ModelVersion;
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
  version,
}: Props) => {
  const handleDownloadClick = useCallback(() => onDownload?.(), [onDownload]);

  const handleDeleteClick = useCallback(() => onDelete?.(), [onDelete]);

  const { canDeleteModelVersion } = usePermissions();

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

    const menuItems: MenuProps['items'] = [{ key: MenuKey.Download, label: 'Download' }];
    if (canDeleteModelVersion({ modelVersion: version })) {
      menuItems.push({ danger: true, key: MenuKey.DeleteVersion, label: 'Deregister Version' });
    }

    return { items: menuItems, onClick: onItemClick };
  }, [handleDeleteClick, handleDownloadClick, canDeleteModelVersion, version]);

  return children ? (
    <Dropdown
      menu={ModelVersionActionMenu}
      placement="bottomLeft"
      trigger={trigger ?? ['contextMenu', 'click']}
      onOpenChange={onVisibleChange}>
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
        <Button type="text" onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </Button>
      </Dropdown>
    </div>
  );
};

export default ModelVersionActionDropdown;
