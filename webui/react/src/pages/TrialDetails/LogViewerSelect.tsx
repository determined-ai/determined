import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import { alphaNumericSorter } from 'hew/internal/functions';
import { LogLevelFromApi } from 'hew/internal/types';
import Row from 'hew/Row';
import Select, { Option, SelectValue } from 'hew/Select';
import { isArray } from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { throttle } from 'throttle-debounce';

interface Props {
  onChange?: (filters: Filters) => void;
  onReset?: () => void;
  options: Filters;
  showSearch: boolean;
  values: Filters;
  onClickSearch?: () => void;
  searchOn?: boolean;
}

export interface Filters {
  agentIds?: string[];
  allocationIds?: string[];
  containerIds?: string[];
  levels?: LogLevelFromApi[];
  rankIds?: number[];
  searchText?: string;
  enableRegex?: boolean;
  // sources?: string[],
  // stdtypes?: string[],
}

export const ARIA_LABEL_RESET = 'Reset';

export const LABELS: Record<keyof Filters, string> = {
  agentIds: 'Agents',
  allocationIds: 'Allocations',
  containerIds: 'Containers',
  enableRegex: 'Regex',
  levels: 'Levels',
  rankIds: 'Ranks',
  searchText: 'Searches',
};

const LogViewerSelect: React.FC<Props> = ({
  onChange,
  onReset,
  options,
  showSearch,
  onClickSearch,
  searchOn,
  values,
}: Props) => {
  const [filters, setFilters] = useState<Filters>(values);

  const selectOptions = useMemo(() => {
    const { agentIds, allocationIds, containerIds, rankIds } = options;
    return {
      ...options,
      agentIds: agentIds?.sort(alphaNumericSorter),
      allocationIds: allocationIds?.sort(alphaNumericSorter),
      containerIds: containerIds?.sort(alphaNumericSorter),
      levels: Object.entries(LogLevelFromApi)
        .filter((entry) => entry[1] !== LogLevelFromApi.Unspecified)
        .map(([key, value]) => ({ label: key, value })),
      rankIds: rankIds ? [-1].concat(rankIds).sort(alphaNumericSorter) : [-1],
    };
  }, [options]);

  const moreThanOne = useMemo(() => {
    return Object.keys(selectOptions).reduce(
      (acc, key) => {
        const filterKey = key as keyof Filters;
        if (filterKey === 'enableRegex') return acc;
        const options = selectOptions[filterKey];

        // !! casts `undefined` into the boolean value of `false`.
        acc[filterKey] = !!(options && options.length > 1);

        return acc;
      },
      {} as Record<keyof Filters, boolean>,
    );
  }, [selectOptions]);

  const isResetShown = useMemo(() => {
    if (values.searchText) return true;

    const keys = Object.keys(selectOptions);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i] as keyof Filters;

      const value = values[key];
      if (key === 'enableRegex' && value) {
        return true;
      }
      if (value && isArray(value) && value.length !== 0) return true;
    }

    return false;
  }, [selectOptions, values]);

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
    return () => {
      throttledChangeFilter.cancel();
    };
  }, [throttledChangeFilter]);

  const handleChange = useCallback(
    (key: keyof Filters, caster: NumberConstructor | StringConstructor) => (value: SelectValue) => {
      setFilters((prev) => {
        const newF = {
          ...prev,
          [key]: (value as Array<string>).map((item) => caster(item)),
        };
        throttledChangeFilter(newF);
        return newF;
      });
    },
    [throttledChangeFilter],
  );

  const handleSearch = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFilters((prev) => {
        const newF = { ...prev, searchText: e.target.value };
        throttledChangeFilter(newF);
        return newF;
      });
    },
    [throttledChangeFilter],
  );

  const handleReset = useCallback(() => {
    setFilters({});
    onReset?.();
    throttledChangeFilter({});
  }, [onReset, throttledChangeFilter]);

  return (
    <Row>
      {showSearch && (
        <Input placeholder="Search Logs..." value={filters.searchText} onChange={handleSearch} />
      )}
      {onClickSearch && (
        <Button onClick={onClickSearch}>
          <Icon name={searchOn ? 'close' : 'search'} title="search" />
        </Button>
      )}
      {moreThanOne.allocationIds && (
        <Select
          disableTags
          mode="multiple"
          placeholder={`All ${LABELS.allocationIds}`}
          value={filters.allocationIds}
          width={120}
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
          width={120}
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
          width={120}
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
          width={120}
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
        width={120}
        onChange={handleChange('levels', String)}>
        {selectOptions?.levels.map((level) => (
          <Option key={level.value} value={level.value}>
            {level.label}
          </Option>
        ))}
      </Select>
      {isResetShown && <Button onClick={handleReset}>{ARIA_LABEL_RESET}</Button>}
    </Row>
  );
};

export default LogViewerSelect;
