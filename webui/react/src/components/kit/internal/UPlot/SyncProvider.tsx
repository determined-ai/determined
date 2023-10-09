import { observable } from 'micro-observables';
import React, { createContext, useContext, useMemo } from 'react';
import uPlot, { AlignedData } from 'uplot';

import { generateUUID } from 'components/kit/internal/functions';
import { XAxisDomain } from 'components/kit/internal/types';

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
  axis: XAxisDomain | undefined;
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

  updateDataBounds(data: AlignedData, axis?: XAxisDomain) {
    const xValues = data[0];
    const lastIdx = xValues.length - 1;
    const chartMin = xValues[0];
    const chartMax = xValues[lastIdx];

    if (chartMin === undefined || chartMax === undefined) return;

    this.bounds.update((b) => {
      const resetAxis = axis !== this.axis && !!axis && !!this.axis;
      if (axis) {
        this.axis = axis;
      }

      const previousMin =
        b.dataBounds?.min !== undefined && isFinite(b.dataBounds?.min) && !resetAxis
          ? b.dataBounds?.min
          : chartMin;

      const previousMax =
        b.dataBounds?.max !== undefined && isFinite(b.dataBounds?.max) && !resetAxis
          ? b.dataBounds?.max
          : chartMax;

      const dataMin = Math.min(previousMin, chartMin);
      const dataMax = Math.max(previousMax, chartMax);

      const width = dataMax - dataMin;
      const margin = 0.02 * width;

      const unzoomedMin = width > 0 ? dataMin - margin : Math.min(dataMin, 0);
      let unzoomedMax = width > 0 ? dataMax + margin : 2 * dataMax;
      if (unzoomedMin === unzoomedMax) {
        // for single point at x=0
        unzoomedMax = 1;
      }

      return {
        ...b,
        dataBounds: { max: dataMax, min: dataMin },
        unzoomedBounds: { max: unzoomedMax, min: unzoomedMin },
        zoomBounds: resetAxis ? null : b.zoomBounds,
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
