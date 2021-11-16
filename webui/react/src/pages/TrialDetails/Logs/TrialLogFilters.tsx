import { Button, Select, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import MultiSelect from 'components/MultiSelect';

const { Option } = Select;

interface Props {
  onChange?: (filters: Filters) => void;
  onReset?: () => void;
  options: Filters;
  values: Filters;
}

export enum LogLevelFromApi {
  Unspecified = 'LOG_LEVEL_UNSPECIFIED',
  Trace = 'LOG_LEVEL_TRACE',
  Debug = 'LOG_LEVEL_DEBUG',
  Info = 'LOG_LEVEL_INFO',
  Warning = 'LOG_LEVEL_WARNING',
  Error = 'LOG_LEVEL_ERROR',
  Critical = 'LOG_LEVEL_CRITICAL',
}

export interface Filters {
  agentIds?: string[],
  containerIds?: string[],
  levels?: LogLevelFromApi[],
  rankIds?: number[],
  sources?: string[],
  stdtypes?: string[],
}

const TrialLogFilters: React.FC<Props> = ({ onChange, onReset, options, values }: Props) => {
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

  const handleReset = useCallback(() => onReset?.(), [ onReset ]);

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
            style={{ width: 150 }}
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
