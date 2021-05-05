import React, { useState } from 'react';

import ArchiveSelectFilter from 'components/ArchiveSelectFilter';
import { ArchiveFilters } from 'types';

export default {
  component: ArchiveSelectFilter,
  title: 'ArchiveSelectFilter',
};

export const Default = (): React.ReactNode => {
  const [ currentValue, setCurrentValue ] = useState<ArchiveFilters>('unarchived');
  return <ArchiveSelectFilter
    value={currentValue}
    onChange={(newValue) => setCurrentValue(newValue)} />;
};
