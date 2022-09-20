import { Button } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { DEFAULT_COLUMNS } from 'pages/ProjectDetails.settings';
import { generateAlphaNumeric } from 'shared/utils/string';

import useModalColumnsCustomize from './useModalColumnsCustomize';

export default {
  component: useModalColumnsCustomize,
  title: 'CustomizeColumnModal',
};

export const Default = (): React.ReactNode => {
  const columns = useMemo(() => {
    const arr = [...DEFAULT_COLUMNS] as string[];
    for (let i = 0; i < 50; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  const { modalOpen } = useModalColumnsCustomize({
    columns,
    defaultVisibleColumns: DEFAULT_COLUMNS,
  });

  const openModal = useCallback(() => {
    modalOpen({ initialVisibleColumns: DEFAULT_COLUMNS });
  }, [modalOpen]);

  return <Button onClick={openModal}>Columns</Button>;
};

export const LongList = (): React.ReactNode => {
  const columns = useMemo(() => {
    const arr = [...DEFAULT_COLUMNS] as string[];
    for (let i = 0; i < 50000; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  const { modalOpen } = useModalColumnsCustomize({
    columns,
    defaultVisibleColumns: DEFAULT_COLUMNS,
  });

  const openModal = useCallback(() => {
    modalOpen({ initialVisibleColumns: DEFAULT_COLUMNS });
  }, [modalOpen]);

  return <Button onClick={openModal}>Columns</Button>;
};
