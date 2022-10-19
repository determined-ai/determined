import React, { useMemo } from 'react';

import { DEFAULT_COLUMNS } from 'pages/ExperimentList.settings';
import { generateAlphaNumeric } from 'shared/utils/string';

import Transfer from './Transfer';

export default {
  component: Transfer,
  title: 'Determined/Transfer',
};

export const Default = (): React.ReactNode => {
  const columns = useMemo(() => {
    const arr = [...DEFAULT_COLUMNS] as string[];
    for (let i = 0; i < 50; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  return (
    <div style={{ width: 400 }}>
      <Transfer defaultTargetEntries={DEFAULT_COLUMNS} entries={columns} />
    </div>
  );
};

export const LongList = (): React.ReactNode => {
  const columns = useMemo(() => {
    const arr = [...DEFAULT_COLUMNS] as string[];
    for (let i = 0; i < 50000; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  return (
    <div style={{ width: 400 }}>
      <Transfer defaultTargetEntries={DEFAULT_COLUMNS} entries={columns} />
    </div>
  );
};
