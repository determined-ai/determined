import { Button } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { DEFAULT_COLUMNS } from 'pages/ExperimentList.settings';
import { generateAlphaNumeric } from 'utils/string';

import useCustomizeColumnsModal from './useCustomizeColumnsModal';

export default {
  component: useCustomizeColumnsModal,
  title: 'CustomizeColumnModal',
};

export const Default = (): React.ReactNode => {
  const { showModal } = useCustomizeColumnsModal();

  const columns = useMemo(() => {
    const arr = [ ...DEFAULT_COLUMNS ];
    for (let i = 0; i < 50; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  const openModal = useCallback(() => {
    showModal({
      columns: columns,
      defaultVisibleColumns: DEFAULT_COLUMNS,
      initialVisibleColumns: DEFAULT_COLUMNS,
    });
  }, [ columns, showModal ]);

  return (
    <Button onClick={openModal}>Columns</Button>
  );
};

export const LongList = (): React.ReactNode => {
  const { showModal } = useCustomizeColumnsModal();

  const columns = useMemo(() => {
    const arr = [ ...DEFAULT_COLUMNS ];
    for (let i = 0; i < 50000; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  const openModal = useCallback(() => {
    showModal({
      columns: columns,
      defaultVisibleColumns: DEFAULT_COLUMNS,
      initialVisibleColumns: DEFAULT_COLUMNS,
    });
  }, [ columns, showModal ]);

  return (
    <Button onClick={openModal}>Columns</Button>
  );
};
