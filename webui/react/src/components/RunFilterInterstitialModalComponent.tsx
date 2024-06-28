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
import { observable } from 'utils/observable';

export type CloseReason = 'has_search_runs' | 'no_search_runs' | 'failed' | 'close' | 'manual';

export interface Props {
  projectId?: number;
  selection: SelectionType;
  filterFormSet: FilterFormSetWithoutId;
}

export interface ControlledModalRef {
  open: () => Promise<CloseReason>;
  close: (reason?: CloseReason) => void;
}

/**
 * Modal component for checking selections for runs that are part of a search.
 * is essentially a single purpose interstitial modal component. Because it
 * wraps a modal and the intended use is within a user flow, this component does
 * not use the `useModal` hook. instead, it exposes control via ref. the `open`
 * method of the ref returns a promise that resolves when the modal is closed
 * with the reason why the modal closed.
 *
 */
export const RunFilterInterstitialModalComponent = forwardRef<ControlledModalRef, Props>(
  ({ projectId, selection, filterFormSet }: Props, ref): JSX.Element => {
    const InterstitialModal = useModal(InterstitialModalComponent);
    const [isOpen, setIsOpen] = useState<boolean>(false);
    const closeController = useRef(new AbortController());
    const lifecycleObservable = useRef(observable<CloseReason | null>(null));

    const { close: internalClose, open: internalOpen } = InterstitialModal;

    const open = async () => {
      internalOpen();
      setIsOpen(true);
      const closeReason = await lifecycleObservable.current.toPromise();
      if (closeReason === null) {
        // this promise should never reject -- toPromise only resolves when the
        // value changes, and no code sets the observavble to null
        return Promise.reject();
      }
      return closeReason;
    };

    const close = useCallback(
      (reason: CloseReason = 'manual') => {
        setIsOpen(false);
        // encourage render with isOpen to false before closing to prevent
        // firing onCloseAction twice
        setTimeout(() => internalClose('close'), 0);
        closeController.current.abort();
        closeController.current = new AbortController();
        lifecycleObservable.current.set(reason);
        lifecycleObservable.current = observable(null);
      },
      [internalClose],
    );

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
                    conjunction: 'and',
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
                columnName: 'searcherType',
                kind: 'field',
                location: 'LOCATION_TYPE_RUN',
                operator: '!=',
                type: 'COLUMN_TYPE_TEXT',
                value: 'single',
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
              limit: -2,
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
        if (reason === 'ok') {
          return selectionHasSearchRuns.forEach((bool) => {
            const fixedReason = bool ? 'has_search_runs' : 'no_search_runs';
            close(fixedReason);
          });
        }
        close(reason);
      },
      [close, selectionHasSearchRuns],
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
