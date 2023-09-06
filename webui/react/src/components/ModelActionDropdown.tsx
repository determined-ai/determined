import React, { useCallback } from 'react';

import DeleteModelModal from 'components/DeleteModelModal';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import { useModal } from 'components/kit/Modal';
import ModelMoveModal from 'components/ModelMoveModal';
import usePermissions from 'hooks/usePermissions';
import { archiveModel, unarchiveModel } from 'services/api';
import { ModelItem } from 'types';

export const ModelActionMenuKey = {
  DeleteModel: 'delete-model',
  MoveModel: 'move-model',
  SwitchArchived: 'switch-archived',
} as const;

interface Props {
  children: React.ReactNode;
  record: ModelItem;
}

export const ModelActionDropdown: React.FC<Props> = ({ children, record: model }: Props) => {
  const deleteModelModal = useModal(DeleteModelModal);
  const modelMoveModal = useModal(ModelMoveModal);

  const { canDeleteModel, canModifyModel } = usePermissions();
  const canDelete = canDeleteModel({ model });
  const canModify = canModifyModel({ model });

  const switchArchived = useCallback(async () => {
    if (model.archived) {
      await unarchiveModel({ modelName: model.name });
    } else {
      await archiveModel({ modelName: model.name });
    }
  }, [model.archived, model.name]);

  const handleDropdown = (key: string) => {
    switch (key) {
      case ModelActionMenuKey.DeleteModel:
        deleteModelModal.open();
        break;
      case ModelActionMenuKey.MoveModel:
        modelMoveModal.open();
        break;
      case ModelActionMenuKey.SwitchArchived:
        switchArchived();
        break;
    }
  };

  const ModelActionMenu = useCallback(
    (record: ModelItem) => {
      const menuItems: MenuItem[] = [];
      if (canModify) {
        menuItems.push({
          key: ModelActionMenuKey.SwitchArchived,
          label: record.archived ? 'Unarchive' : 'Archive',
        });
        if (!record.archived) {
          menuItems.push({ key: ModelActionMenuKey.MoveModel, label: 'Move' });
        }
      }
      if (canDelete) {
        menuItems.push({
          danger: true,
          key: ModelActionMenuKey.DeleteModel,
          label: 'Delete Model',
        });
      }

      return menuItems;
    },
    [canModify, canDelete],
  );

  return (
    <>
      <Dropdown
        isContextMenu
        menu={ModelActionMenu(model)}
        onClick={(key: string) => handleDropdown(key)}>
        {children}
      </Dropdown>
      {model && <deleteModelModal.Component model={model} />}
      {model && <modelMoveModal.Component model={model} />}
    </>
  );
};
