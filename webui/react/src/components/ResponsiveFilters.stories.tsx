import { Select } from 'antd';
import React from 'react';

import ResponsiveFilters from './ResponsiveFilters';
import SelectFilter from './SelectFilter';

const { Option } = Select;

export default {
  component: ResponsiveFilters,
  parameters: { layout: 'centered' },
  title: 'ResponsiveFilters',
};

const options = new Array(10).fill(null).map((_, index) => (
  <Option key={index} value={index}>Option {index}</Option>
));

export const Default = (): React.ReactNode => (
  <div>
    <div style={{
      color: '#999999',
      fontSize: 14,
      fontStyle: 'italic',
      marginBottom: 10,
      textAlign: 'center',
      width: '100%',
    }}>
      Reduce this browser width to see the responsive effect.
    </div>
    <ResponsiveFilters>
      {new Array(4).fill(null).map((_, index) => (
        <SelectFilter key={index} label={`Filter ${index}`} value={0}>{options}</SelectFilter>
      ))}
    </ResponsiveFilters>
  </div>
);
