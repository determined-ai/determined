import { useModal } from 'hew/Modal';
import { Failed, NotLoaded } from 'hew/utils/loadable';
import { forwardRef, useCallback, useImperativeHandle, useRef, useState } from 'react';

import { FilterFormSetWithoutId, Operator } from 'components/FilterForm/components/type';
import InterstitialModalComponent, {
  onInterstitialCloseActionType,
} from 'components/InterstitialModalComponent';
import { SelectionType } from 'components/Searches/Searches.settings';
import { useAsync } from 'hooks/useAsync';
import { searchRuns } from 'services/api';
import { DetError } from 'utils/error';
import mergeAbortControllers from 'utils/mergeAbortControllers';

export interface Props {
  projectId?: number;
  selection: SelectionType;
  filterFormSet: FilterFormSetWithoutId;
  onCloseAction: (reason: 'has_search_runs' | 'no_search_runs' | 'failed' | 'close') => void;
}

export interface ControlledModalRef {
  open: () => void;
  close: () => void;
}

/**
 * Modal component for checking selections for runs that are part of a search.
 * is essentially a single purpose interstitial modal component. instead of firing `ok`, fires `has_search_runs` or `no_search_runs` to the close handler
 *
 */
export const RunFilterInterstitialModalComponent = forwardRef<ControlledModalRef, Props>(
  ({ projectId, selection, filterFormSet, onCloseAction }: Props, ref): JSX.Element => {
    const InterstitialModal = useModal(InterstitialModalComponent);
    const [isOpen, setIsOpen] = useState<boolean>(false);
    const closeController = useRef(new AbortController());

    const { close: internalClose, open: internalOpen } = InterstitialModal;

    const open = () => {
      setIsOpen(true);
      internalOpen();
    };

    const close = useCallback(() => {
      setIsOpen(false);
      // encourage render with isOpen to false before closing to prevent
      // firing onCloseAction twice
      setTimeout(() => internalClose('close'), 0);
      closeController.current.abort();
      closeController.current = new AbortController();
    }, [internalClose]);

    useImperativeHandle(ref, () => ({ close, open }));

    const selectionHasSearchRuns = useAsync(
      async (canceler) => {
        if (!isOpen) return NotLoaded;
        const mergedCanceler = mergeAbortControllers(canceler, closeController.current);
        const idToFilter = (operator: Operator, id: number) =>
          ({
            columnName: 'id',
            kind: 'field',
            location: 'LOCATION_TYPE_RUN',
            operator,
            type: 'COLUMN_TYPE_NUMBER',
            value: id,
          }) as const;
        const filterGroup: FilterFormSetWithoutId['filterGroup'] =
          selection.type === 'ALL_EXCEPT'
            ? {
                children: [
                  filterFormSet.filterGroup,
                  {
                    children: selection.exclusions.map(idToFilter.bind(this, '!=')),
                    conjunction: 'or',
                    kind: 'group',
                  },
                ],
                conjunction: 'and',
                kind: 'group',
              }
            : {
                children: selection.selections.map(idToFilter.bind(this, '=')),
                conjunction: 'or',
                kind: 'group',
              };
        const filter: FilterFormSetWithoutId = {
          ...filterFormSet,
          filterGroup: {
            children: [
              filterGroup,
              {
                columnName: 'numTrials',
                kind: 'field',
                location: 'LOCATION_TYPE_EXPERIMENT',
                operator: '>',
                type: 'COLUMN_TYPE_NUMBER',
                value: 1,
              } as const,
            ],
            conjunction: 'and',
            kind: 'group',
          },
        };
        try {
          const results = await searchRuns(
            {
              filter: JSON.stringify(filter),
              limit: 0,
              projectId,
            },
            { signal: mergedCanceler.signal },
          );

          return (results.pagination.total || 0) > 0;
        } catch (e) {
          if (!mergedCanceler.signal.aborted) {
            return Failed(e instanceof Error ? e : new DetError(e));
          }
          return NotLoaded;
        }
      },
      [selection, filterFormSet, projectId, isOpen],
    );

    const interstitialClose: onInterstitialCloseActionType = useCallback(
      (reason) => {
        close();
        if (reason === 'ok') {
          return selectionHasSearchRuns.forEach((bool) => {
            onCloseAction(bool ? 'has_search_runs' : 'no_search_runs');
          });
        }
        onCloseAction(reason);
      },
      [close, onCloseAction, selectionHasSearchRuns],
    );

    return (
      <InterstitialModal.Component
        loadableData={selectionHasSearchRuns}
        onCloseAction={interstitialClose}
      />
    );
  },
);

export default RunFilterInterstitialModalComponent;
