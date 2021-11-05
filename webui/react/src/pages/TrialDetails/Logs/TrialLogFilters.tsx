import { Button, Select, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import MultiSelect from 'components/MultiSelect';

const { Option } = Select;

interface Props {
  onChange?: (filters: Filters) => void;
  options: Filters;
  values: Filters;
}

enum LogLevelFromApi {
  Unspecified = 'LOG_LEVEL_UNSPECIFIED',
  Trace = 'LOG_LEVEL_TRACE',
  Debug = 'LOG_LEVEL_DEBUG',
  Info = 'LOG_LEVEL_INFO',
  Warning = 'LOG_LEVEL_WARNING',
  Error = 'LOG_LEVEL_ERROR',
  Critical = 'LOG_LEVEL_CRITICAL',
}

export interface Filters {
  agentIds?: Array<string>,
  containerIds?: Array<string>,
  levels?: Array<LogLevelFromApi>,
  rankIds?: Array<number>,
  sources?: Array<string>,
  stdtypes?: Array<string>,
}

const TrialLogFilters: React.FC<Props> = ({ onChange, options, values }: Props) => {
  const selectOptions = useMemo(() => {
    return {
      ...options,
      levels: Object.entries(LogLevelFromApi)
        .filter(entry => entry[1] !== LogLevelFromApi.Unspecified)
        .map(([ key, value ]) => ({ label: key, value })),
    };
  }, [ options ]);

  const handleChange = useCallback((
    key: keyof Filters,
    caster: NumberConstructor | StringConstructor,
  ) => (value: SelectValue) => {
    onChange?.({
      ...values,
      [key]: (value as Array<string>).map(item => caster(item)),
    });
  }, [ onChange, values ]);

  const handleReset = useCallback(() => onChange?.({}), [ onChange ]);

  return (
    <>
      <Space>
        {selectOptions?.agentIds?.length !== 0 && (
          <MultiSelect
            itemName="Agent"
            value={values.agentIds}
            onChange={handleChange('agentIds', String)}>
            {selectOptions?.agentIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        {selectOptions?.containerIds?.length !== 0 && (
          <MultiSelect
            itemName="Container"
            value={values.containerIds}
            onChange={handleChange('containerIds', String)}>
            {selectOptions?.containerIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        {selectOptions?.rankIds?.length !== 0 && (
          <MultiSelect
            itemName="Rank"
            value={values.rankIds}
            onChange={handleChange('rankIds', Number)}>
            {selectOptions?.rankIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        <MultiSelect
          itemName="Level"
          value={values.levels}
          onChange={handleChange('levels', String)}>
          {selectOptions?.levels.map((level) => (
            <Option key={level.value} value={level.value}>{level.label}</Option>
          ))}
        </MultiSelect>
        <Button onClick={handleReset}>Reset</Button>
      </Space>
    </>
  );
};

export default TrialLogFilters;
