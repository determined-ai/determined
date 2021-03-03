import React, { useCallback, useState } from 'react';

import GridListRadioGroup, { GridListView } from './GridListRadioGroup';

export default {
  component: GridListRadioGroup,
  parameters: { layout: 'centered' },
  title: 'GridListRadioGroup',
};

export const Default = (): React.ReactNode => {
  const [ view, setView ] = useState<GridListView>(GridListView.Grid);

  const handleChange = useCallback((value: GridListView) => setView(value), []);

  return <GridListRadioGroup value={view} onChange={handleChange} />;
};
