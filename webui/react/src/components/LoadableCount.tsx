import Button from 'hew/Button';
import { HandleSelectionChangeType } from 'hew/DataGrid/DataGrid';
import { Loadable } from 'hew/utils/loadable';
import { useMemo } from 'react';

import useMobile from 'hooks/useMobile';
import { pluralizer } from 'utils/string';

import css from './LoadableCount.module.scss';

export type SelectionAction = 'SELECT_ALL' | 'CLEAR_SELECTION' | 'NONE';
interface Props {
  total: Loadable<number>;
  labelSingular: string;
  labelPlural: string;
  selectedCount: number;
  selectionAction: SelectionAction;
  handleSelectionChange: HandleSelectionChangeType;
}

const LoadableCount: React.FC<Props> = ({
  total,
  labelPlural,
  labelSingular,
  selectedCount,
  selectionAction,
  handleSelectionChange,
}: Props) => {
  const isMobile = useMobile();

  const selectionLabel = useMemo(() => {
    return Loadable.match(total, {
      Failed: () => null,
      Loaded: (loadedTotal) => {
        let label = `${loadedTotal.toLocaleString()} ${pluralizer(
          loadedTotal,
          labelSingular.toLowerCase(),
          labelPlural.toLowerCase(),
        )}`;

        if (selectedCount) {
          label = `${selectedCount.toLocaleString()} of ${label} selected`;
        }

        return label;
      },
      NotLoaded: () => `Loading ${labelPlural.toLowerCase()}...`,
    });
  }, [labelPlural, labelSingular, total, selectedCount]);

  const actualSelectAll = useMemo(() => {
    return Loadable.match(total, {
      _: () => null,
      Loaded: () => {
        switch (selectionAction) {
          case 'SELECT_ALL': {
            const onClick = () => handleSelectionChange('add-all');
            return (
              <Button data-test="select-all" type="text" onClick={onClick}>
                Select all {labelPlural} in table
              </Button>
            );
          }
          case 'CLEAR_SELECTION': {
            const onClick = () => handleSelectionChange('remove-all');
            return (
              <Button data-test="clear-selection" type="text" onClick={onClick}>
                Clear Selection
              </Button>
            );
          }
          case 'NONE': {
            return null;
          }
        }
      },
    });
  }, [labelPlural, handleSelectionChange, selectionAction, total]);

  if (!isMobile) {
    return (
      <>
        <span className={css.base} data-test="count">
          {selectionLabel}
        </span>
        {actualSelectAll}
      </>
    );
  } else {
    return null;
  }
};

export default LoadableCount;
