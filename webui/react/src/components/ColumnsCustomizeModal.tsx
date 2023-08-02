import React, { useRef, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import Transfer from 'components/Transfer';
import handleError from 'utils/error';

interface Props {
  columns: string[];
  defaultVisibleColumns: string[];
  initialVisibleColumns?: string[];
  onSave?: (columns: string[]) => void;
}

const ColumnsCustomizeModalComponent: React.FC<Props> = ({
  columns,
  defaultVisibleColumns,
  initialVisibleColumns,
  onSave,
}: Props) => {
  const columnList = useRef(columns).current; // This is only to prevent rerendering
  const [visibleColumns, setVisibleColumns] = useState<string[]>(
    initialVisibleColumns ?? defaultVisibleColumns,
  );

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        handleError,
        handler: async () => {
          return await onSave?.(visibleColumns);
        },
        text: 'Save',
      }}
      title="Customize Columns">
      <Transfer
        defaultTargetEntries={defaultVisibleColumns}
        entries={columnList}
        initialTargetEntries={visibleColumns}
        persistentEntries={['id', 'name']}
        sourceListTitle="Hidden"
        targetListTitle="Visible"
        onChange={setVisibleColumns}
      />
    </Modal>
  );
};

export default ColumnsCustomizeModalComponent;
