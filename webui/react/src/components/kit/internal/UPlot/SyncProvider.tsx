import { observable } from 'micro-observables';
import React, { createContext, useContext, useMemo } from 'react';
import uPlot, { AlignedData } from 'uplot';

import { generateUUID } from 'components/kit/internal/functions';

type Bounds = {
  dataBounds: {
    max: number;
    min: number;
  } | null;
  unzoomedBounds: {
    max: number;
    min: number;
  } | null;
  zoomBounds: {
    max: number;
    min: number;
  } | null;
};

class SyncService {
  bounds = observable<Bounds>({ dataBounds: null, unzoomedBounds: null, zoomBounds: null });

  pubSub: uPlot.SyncPubSub;

  activeBounds = this.bounds.select((b) => b?.zoomBounds ?? b?.unzoomedBounds);
  key: string;

  constructor(syncKey?: string) {
    this.key = syncKey ?? generateUUID();
    this.pubSub = uPlot.sync(this.key);
    this.activeBounds.subscribe((activeBounds) => {
      if (!activeBounds) return;
      const { min, max } = activeBounds;
      this.pubSub.plots.forEach((u) => {
        u.setScale('x', { max, min });
      });
    });
  }

  syncChart(chart: uPlot) {
    const activeBounds = this?.activeBounds.get();
    if (activeBounds) chart.setScale('x', activeBounds);
  }

  resetZoom() {
    this.bounds.update((b) => ({ ...b, zoomBounds: null }));
  }

  setZoom(min: number, max: number) {
    this.bounds.update((b) => ({ ...b, zoomBounds: { max, min } }));
  }

  updateDataBounds(data: AlignedData) {
    const xValues = data[0];
    const lastIdx = xValues.length - 1;
    const dataMin = xValues[0];
    const dataMax = xValues[lastIdx];

    if (dataMin === undefined || dataMax === undefined) return;

    this.bounds.update((b) => {
      let max = Math.max(b.dataBounds?.max ?? dataMax, dataMax);
      let min = Math.min(b.dataBounds?.min ?? dataMin, dataMin);
      if (min === max) {
        if (max < 0) {
          min = max;
          max = 0;
        } else {
          min = 0;
        }
      }
      return {
        ...b,
        dataBounds: { max, min },
        unzoomedBounds: { max, min },
      };
    });
  }
}

interface Props {
  children: React.ReactNode;
  // pass a new key when you want the zoom to be reset,
  // e.g. when changing the x-axis. by default it will
  // reset when the component remounts
  syncKey?: string;
}

const SyncContext = createContext<SyncService | null>(null);

export const SyncProvider: React.FC<Props> = ({ syncKey, children }) => {
  const syncService = useMemo(() => new SyncService(syncKey), [syncKey]);

  return <SyncContext.Provider value={syncService}>{children}</SyncContext.Provider>;
};

export const useChartSync = (): {
  options: Partial<uPlot.Options>;
  syncService: SyncService;
} => {
  const syncProviderService = useContext(SyncContext);

  const syncService = useMemo(
    () => syncProviderService ?? new SyncService(),
    [syncProviderService],
  );

  const options = useMemo(() => {
    const syncKey = syncService?.pubSub.key;
    const syncScales: [string, null] = ['x', null];
    return {
      cursor: {
        bind: {
          dblclick: () => {
            return () => {
              syncService?.resetZoom();
              return null;
            };
          },
        },
        drag: { dist: 5, setScale: false, uni: 10, x: true, y: false },
        sync: {
          key: syncKey,
          scales: syncScales,
          setSeries: false,
        },
      },

      hooks: {
        init: [syncService.syncChart],
        ready: [syncService.syncChart],
        setData: [(chart: uPlot) => syncService.updateDataBounds(chart.data)],
        setSelect: [
          (chart: uPlot) => {
            const min = chart.posToVal(chart.select.left, 'x');
            const max = chart.posToVal(chart.select.left + chart.select.width, 'x');
            syncService?.setZoom(min, max);
            chart.setSelect({ height: 0, left: 0, top: 0, width: 0 }, false);
          },
        ],
      },
    };
  }, [syncService]);

  return { options, syncService };
};
