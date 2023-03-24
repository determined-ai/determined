import React, { useEffect, useRef, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import Transfer from 'components/Transfer';
import usePrevious from 'shared/hooks/usePrevious';
import { isEqual } from 'shared/utils/data';

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
  const prevInitVisibleColumns = usePrevious(initialVisibleColumns, defaultVisibleColumns);
  const [visibleColumns, setVisibleColumns] = useState<string[]>(
    initialVisibleColumns ?? defaultVisibleColumns,
  );

  useEffect(() => {
    // If you travel between pages that both use this hook `visibleColumns` doesn't reinitialize.
    // That means that when you open this modal on the second page it will have the value of `visibleColumns` from the first page.
    // This useEffect makes sure that if the value of `initialVisibleColumns` prop changes that
    // `visibleColumns` will be set to the new value.
    if (!isEqual(initialVisibleColumns, prevInitVisibleColumns))
      setVisibleColumns(initialVisibleColumns ?? defaultVisibleColumns);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialVisibleColumns]);
  return (
    <Modal
      cancel
      size="medium"
      submit={{
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
        sourceListTitle="Hidden"
        targetListTitle="Visible"
        onChange={setVisibleColumns}
      />
    </Modal>
  );
};

export default ColumnsCustomizeModalComponent;
