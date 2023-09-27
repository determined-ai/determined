import React, { useCallback, useMemo } from 'react';

import css from 'components/ActionDropdown/ActionDropdown.module.scss';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import ModelDownloadModal from 'components/ModelDownloadModal';
import ModelVersionDeleteModal from 'components/ModelVersionDeleteModal';
import usePermissions from 'hooks/usePermissions';
import { ModelVersion } from 'types';

interface Props {
  children?: React.ReactNode;
  className?: string;
  direction?: 'vertical' | 'horizontal';
  isContextMenu?: boolean;
  version: ModelVersion;
}

const MenuKey = {
  DeleteVersion: 'delete-version',
  Download: 'download',
} as const;

const ModelVersionActionDropdown: React.FC<Props> = ({
  children,
  className,
  direction = 'vertical',
  isContextMenu,
  version,
}: Props) => {
  const { canDeleteModelVersion } = usePermissions();

  const modelDownloadModal = useModal(ModelDownloadModal);
  const modelVersionDeleteModal = useModal(ModelVersionDeleteModal);

  const dropdownMenu = useMemo(() => {
    const menuItems: MenuItem[] = [{ key: MenuKey.Download, label: 'Download' }];
    if (canDeleteModelVersion({ modelVersion: version })) {
      menuItems.push({ danger: true, key: MenuKey.DeleteVersion, label: 'Deregister Version' });
    }
    return menuItems;
  }, [canDeleteModelVersion, version]);

  const handleDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case MenuKey.DeleteVersion:
          modelVersionDeleteModal.open();
          break;
        case MenuKey.Download:
          modelDownloadModal.open();
          break;
      }
    },
    [modelDownloadModal, modelVersionDeleteModal],
  );

  return (
    <>
      {children ? (
        <Dropdown isContextMenu={isContextMenu} menu={dropdownMenu} onClick={handleDropdown}>
          {children}
        </Dropdown>
      ) : (
        <div className={[css.base, className].join(' ')} title="Open actions menu">
          <Dropdown menu={dropdownMenu} placement="bottomRight" onClick={handleDropdown}>
            <Button
              icon={<Icon name={`overflow-${direction}`} size="small" title="Action menu" />}
              type="text"
            />
          </Dropdown>
        </div>
      )}
      <modelDownloadModal.Component modelVersion={version} />
      <modelVersionDeleteModal.Component modelVersion={version} />
    </>
  );
};

export default ModelVersionActionDropdown;
