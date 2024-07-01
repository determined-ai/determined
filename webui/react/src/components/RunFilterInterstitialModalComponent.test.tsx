import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { Ref } from 'react';

import { FilterFormSetWithoutId, FormField } from 'components/FilterForm/components/type';
import RunFilterInterstitialModalComponent, {
  CloseReason,
  ControlledModalRef,
  Props,
} from 'components/RunFilterInterstitialModalComponent';
import { ThemeProvider } from 'components/ThemeProvider';

vi.mock('services/api', async () => ({
  ...(await vi.importActual<typeof import('services/api')>('services/api')),
  searchRuns: vi.fn(() =>
    Promise.resolve({
      pagination: {
        total: 0,
      },
    }),
  ),
}));

const { searchRuns } = await import('services/api');
const searchRunsMock = vi.mocked(searchRuns);

const emptyFilterFormSetWithoutId: FilterFormSetWithoutId = {
  filterGroup: {
    children: [],
    conjunction: 'or',
    kind: 'group',
  },
  showArchived: false,
};

const setupTest = (props: Partial<Props> = {}) => {
  const ref: Ref<ControlledModalRef> = { current: null };
  userEvent.setup();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <RunFilterInterstitialModalComponent
          filterFormSet={emptyFilterFormSetWithoutId}
          projectId={1}
          selection={{ selections: [], type: 'ONLY_IN' }}
          {...props}
          ref={ref}
        />
      </ThemeProvider>
    </UIProvider>,
  );

  return {
    ref,
  };
};

describe('RunFilterInterstitialModalComponent', () => {
  beforeEach(() => {
    searchRunsMock.mockRestore();
  });

  it('does not call server until opened', () => {
    const { ref } = setupTest();

    expect(searchRunsMock).not.toBeCalled();
    act(() => {
      ref.current?.open();
    });
    expect(searchRunsMock).toBeCalled();
  });

  it('calls server with filter describing filter selection', () => {
    const expectedFilterGroup: FilterFormSetWithoutId['filterGroup'] = {
      children: [
        {
          columnName: 'experimentName',
          kind: 'field',
          location: 'LOCATION_TYPE_RUN',
          operator: 'contains',
          type: 'COLUMN_TYPE_TEXT',
          value: 'foo',
        },
      ],
      conjunction: 'and',
      kind: 'group',
    };
    const expectedExclusions = [1, 2, 3];
    const { ref } = setupTest({
      filterFormSet: {
        filterGroup: expectedFilterGroup,
        showArchived: true,
      },
      selection: {
        exclusions: expectedExclusions,
        type: 'ALL_EXCEPT',
      },
    });
    act(() => {
      ref.current?.open();
    });

    expect(searchRunsMock).toBeCalled();

    const { lastCall } = vi.mocked(searchRuns).mock;
    const filterFormSetString = lastCall?.[0].filter;
    expect(filterFormSetString).toBeDefined();
    const filterFormSet = JSON.parse(filterFormSetString || '');

    // TODO: is there a better way to test this expectation?
    expect(filterFormSet.showArchived).toBeTruthy();
    const [filterGroup, idFilterGroup] = filterFormSet.filterGroup.children?.[0].children || [];
    expect(filterGroup).toEqual(expectedFilterGroup);

    const idFilters = idFilterGroup.children;
    expect(idFilters.every((f: FormField) => f.operator === '!=')).toBeTruthy();
    expect(idFilters.map((f: FormField) => f.value)).toEqual(expectedExclusions);
  });

  it('calls server with filter describing visual selection', () => {
    const expectedSelection = [1, 2, 3];
    const { ref } = setupTest({
      selection: {
        selections: expectedSelection,
        type: 'ONLY_IN',
      },
    });
    act(() => {
      ref.current?.open();
    });

    expect(searchRunsMock).toBeCalled();

    const { lastCall } = vi.mocked(searchRuns).mock;
    const filterFormSetString = lastCall?.[0].filter;
    expect(filterFormSetString).toBeDefined();
    const filterFormSet = JSON.parse(filterFormSetString || '');

    expect(filterFormSet.showArchived).toBe(false);
    const idFilters = filterFormSet.filterGroup.children?.[0].children || [];
    expect(idFilters.every((f: FormField) => f.operator === '=')).toBe(true);
    expect(idFilters.map((f: FormField) => f.value)).toEqual(expectedSelection);
  });

  it('cancels request when modal is closed via close button', async () => {
    searchRunsMock.mockImplementation((_params, options) => {
      return new Promise((_resolve, reject) => {
        options?.signal?.addEventListener('abort', () => {
          reject();
        });
      });
    });
    const { ref } = setupTest();
    // explicit type here because typescript can't infer that the act function
    // runs imperatively.
    let lifecycle: Promise<CloseReason> | undefined;
    // we don't await the act because we need the render pipeline to flush
    // before we get the close reason back
    act(() => {
      lifecycle = ref.current?.open();
    });
    const closeButton = await screen.findByLabelText('Close');
    await userEvent.click(closeButton);
    const closeReason = await lifecycle;
    expect(closeReason).toBe('close');
  });

  it('closes modal with has_search_runs when it has runs', async () => {
    searchRunsMock.mockImplementation(() =>
      Promise.resolve({
        pagination: {
          total: 1,
        },
        runs: [],
      }),
    );
    const { ref } = setupTest();
    let lifecycle: Promise<CloseReason> | undefined;
    act(() => {
      lifecycle = ref.current?.open();
    });
    const closeReason = await lifecycle;
    expect(closeReason).toBe('has_search_runs');
  });

  it('closes modal with no_search_runs when it lacks runs', async () => {
    searchRunsMock.mockImplementation(() =>
      Promise.resolve({
        pagination: {
          total: 0,
        },
        runs: [],
      }),
    );
    const { ref } = setupTest();
    let lifecycle: Promise<CloseReason> | undefined;
    act(() => {
      lifecycle = ref.current?.open();
    });
    const closeReason = await lifecycle;
    expect(closeReason).toBe('no_search_runs');
  });

  it('closes modal with failed when request errors outside of aborts', async () => {
    searchRunsMock.mockImplementation(() => Promise.reject(new Error('uh oh!')));
    const { ref } = setupTest();
    let lifecycle: Promise<CloseReason> | undefined;
    act(() => {
      lifecycle = ref.current?.open();
    });
    const closeReason = await lifecycle;
    expect(closeReason).toBe('failed');
  });
});
