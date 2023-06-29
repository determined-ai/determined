import { Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { throttle } from 'throttle-debounce';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import { alphaNumericSorter } from 'components/kit/internal/functions';
import { LogLevelFromApi } from 'components/kit/internal/types';
import Select, { Option } from 'components/kit/Select';

interface Props {
  onChange?: (filters: Filters) => void;
  onReset?: () => void;
  options: Filters;
  showSearch: boolean;
  values: Filters;
}

export interface Filters {
  agentIds?: string[];
  allocationIds?: string[];
  containerIds?: string[];
  levels?: LogLevelFromApi[];
  rankIds?: number[];
  searchText?: string;
  // sources?: string[],
  // stdtypes?: string[],
}

export const ARIA_LABEL_RESET = 'Reset';

export const LABELS: Record<keyof Filters, string> = {
  agentIds: 'Agents',
  allocationIds: 'Allocations',
  containerIds: 'Containers',
  levels: 'Levels',
  rankIds: 'Ranks',
  searchText: 'Searches',
};

const LogViewerSelect: React.FC<Props> = ({
  onChange,
  onReset,
  options,
  showSearch,
  values,
}: Props) => {
  const [filters, setFilters] = useState<Filters>(values);

  const selectOptions = useMemo(() => {
    const { agentIds, allocationIds, containerIds, rankIds } = options;
    return {
      ...options,
      agentIds: agentIds?.sortAll(alphaNumericSorter),
      allocationIds: allocationIds?.sortAll(alphaNumericSorter),
      containerIds: containerIds?.sortAll(alphaNumericSorter),
      levels: Object.entries(LogLevelFromApi)
        .filter((entry) => entry[1] !== LogLevelFromApi.Unspecified)
        .map(([key, value]) => ({ label: key, value })),
      rankIds: rankIds ? [-1].concat(rankIds).sortAll(alphaNumericSorter) : [-1],
    };
  }, [options]);

  const moreThanOne = useMemo(() => {
    return Object.keys(selectOptions).reduce((acc, key) => {
      const filterKey = key as keyof Filters;
      const options = selectOptions[filterKey];

      // !! casts `undefined` into the boolean value of `false`.
      acc[filterKey] = !!(options && options.length > 1);

      return acc;
    }, {} as Record<keyof Filters, boolean>);
  }, [selectOptions]);

  const isResetShown = useMemo(() => {
    if (values.searchText) return true;

    const keys = Object.keys(selectOptions);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i] as keyof Filters;
      const value = values[key];
      if (value && value.length !== 0) return true;
    }

    return false;
  }, [selectOptions, values]);

  const handleChange = useCallback(
    (key: keyof Filters, caster: NumberConstructor | StringConstructor) => (value: SelectValue) => {
      setFilters((prev) => ({
        ...prev,
        [key]: (value as Array<string>).map((item) => caster(item)),
      }));
    },
    [],
  );

  const handleSearch = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) =>
      setFilters((prev) => ({ ...prev, searchText: e.target.value })),
    [],
  );

  const handleReset = useCallback(() => {
    setFilters({});
    onReset?.();
  }, [onReset]);

  const throttledChangeFilter = useMemo(
    () =>
      throttle(
        500,
        (f: Filters) => {
          onChange?.(f);
        },
        { noLeading: true },
      ),
    [onChange],
  );

  useEffect(() => {
    if (!filters) return;
    throttledChangeFilter(filters);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters.searchText, throttledChangeFilter]);

  useEffect(() => {
    return () => {
      throttledChangeFilter.cancel();
    };
  }, [throttledChangeFilter]);

  return (
    <>
      <Space>
        {showSearch && (
          <Input placeholder="Search Logs..." value={filters.searchText} onChange={handleSearch} />
        )}
        {moreThanOne.allocationIds && (
          <Select
            disableTags
            mode="multiple"
            placeholder={`All ${LABELS.allocationIds}`}
            value={filters.allocationIds}
            width={150}
            onBlur={() => onChange?.(filters)}
            onChange={handleChange('allocationIds', String)}>
            {selectOptions?.allocationIds?.map((id, index) => (
              <Option key={id || `no-id-${index}`} value={id}>
                {id || 'No Allocation ID'}
              </Option>
            ))}
          </Select>
        )}
        {!!selectOptions?.agentIds?.length && (
          <Select
            disableTags
            mode="multiple"
            placeholder={`All ${LABELS.agentIds}`}
            value={filters.agentIds}
            width={150}
            onBlur={() => onChange?.(filters)}
            onChange={handleChange('agentIds', String)}>
            {selectOptions?.agentIds?.map((id, index) => (
              <Option key={id || `no-id-${index}`} value={id}>
                {id || 'No Agent ID'}
              </Option>
            ))}
          </Select>
        )}
        {moreThanOne.containerIds && (
          <Select
            disableTags
            mode="multiple"
            placeholder={`All ${LABELS.containerIds}`}
            value={filters.containerIds}
            width={150}
            onBlur={() => onChange?.(filters)}
            onChange={handleChange('containerIds', String)}>
            {selectOptions?.containerIds?.map((id, index) => (
              <Option key={id || `no-id-${index}`} value={id}>
                {id || 'No Container ID'}
              </Option>
            ))}
          </Select>
        )}
        {moreThanOne.rankIds && (
          <Select
            disableTags
            mode="multiple"
            placeholder={`All ${LABELS.rankIds}`}
            value={filters.rankIds}
            width={150}
            onBlur={() => onChange?.(filters)}
            onChange={handleChange('rankIds', Number)}>
            {selectOptions?.rankIds?.map((id, index) => (
              <Option key={id ?? `no-id-${index}`} value={id}>
                {id === -1 ? 'No Rank' : id}
              </Option>
            ))}
          </Select>
        )}
        <Select
          disableTags
          mode="multiple"
          placeholder={`All ${LABELS.levels}`}
          value={filters.levels}
          width={150}
          onBlur={() => onChange?.(filters)}
          onChange={handleChange('levels', String)}>
          {selectOptions?.levels.map((level) => (
            <Option key={level.value} value={level.value}>
              {level.label}
            </Option>
          ))}
        </Select>
        {isResetShown && <Button onClick={handleReset}>{ARIA_LABEL_RESET}</Button>}
      </Space>
    </>
  );
};

export default LogViewerSelect;
