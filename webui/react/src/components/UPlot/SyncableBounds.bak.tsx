// @ts-nocheck
import React, {
  createContext,
  MutableRefObject,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import uPlot from 'uplot';

import { distance } from 'utils/chart';

const timestampInMinutes = t => new Date(t).toTimeString().slice(3, 9);

interface SyncableBounds {
  boundsOptions: Partial<uPlot.Options>;
  isZoomed: (zoomed: boolean) => void;
  zoomed: boolean;
}

const SyncContext = createContext();

const getChartMin = (chart: UPlot) => chart.scales.x.min;
const getChartMax = (chart: UPlot) => chart.scales.x.max;
const getDataMax = (chart: UPlot) => chart.data[0][chart.data[0].length - 1];
const getDataMin = (chart: UPlot) => chart.data[0][0];

export const SyncProvider = ({ children }) => {
  const syncRef = useRef(uPlot.sync('x'));
  const [ syncedMin, setSyncedMin ] = useState();
  const [ syncedMax, setSyncedMax ] = useState();

  const [ zoomedMax, setZoomedMax ] = useState();
  const [ zoomedMin, setZoomedMin ] = useState();
  const [ zoomed, setZoomed ] = useState(false);

  const [ signal, setSignal ] = useState(0);
  const fireSignal = useCallback(() => {
    setSignal(prevSignal => prevSignal === 100 ? 0 : prevSignal + 1);
  }, []);

  const minMaxRef = useRef({});

  useEffect(() => {
    minMaxRef.current.max = syncedMax;
    minMaxRef.current.min = syncedMin;
    console.log('min/max', syncedMin, syncedMax);

    if(zoomed) {
      window.max = zoomedMax;
      window.min = zoomedMin ;
      syncRef.current.plots.forEach((chart: uPlot) => {
        if (zoomedMax != null && zoomedMin != null)
          if (getChartMin(chart) !== zoomedMin || getChartMax(chart) !== zoomedMax) {
            console.log('set scale', chart.title);

            chart.setScale('x', { max: zoomedMax, min: zoomedMin });
          }

      });
    } else {
      window.max = syncedMax;
      window.min = syncedMin;
      syncRef.current.plots.forEach((chart: uPlot) => {
        const chartMin = getChartMin(chart);
        const chartMax = getChartMax(chart);
        // console.log('set bounds', chart.title, syncedMin, syncedMax);
        if (chartMin > syncedMin || chartMax < syncedMax) {
          chart.setScale('x', { max: syncedMax, min: syncedMin });
          // chart.redraw();

        }
      });
    }
  }, [ syncedMin, syncedMax, zoomedMin, zoomedMax, zoomed, signal ]);

  const dispatchScaleUpdate = useCallback(
    (chart: uPlot, scaleKey) => {
      if(scaleKey === 'x') {
        console.log('scale update');

      }
    },
    [ setZoomedMin, setZoomedMax ],
  );

  const dataScaleUpdater = useCallback(
    (chart: uPlot) => {
      console.log('data');
      setSyncedMax(prevMax => {
        return Math.max(getDataMax(chart), prevMax ?? Number.MIN_SAFE_INTEGER);
      });
      setSyncedMin(prevMin => Math.min(getDataMin(chart), prevMin ?? Number.MAX_SAFE_INTEGER));
      // fireSignal();

    },
    [ setSyncedMin, setSyncedMax ],
  );

  return (
    <SyncContext.Provider
      value={{
        dataScaleUpdater,
        dispatchScaleUpdate,
        fireSignal,
        minMaxRef,
        setZoomed,
        setZoomedMax,
        setZoomedMin,
        syncRef,
        zoomed,
      }}>
      {children}
    </SyncContext.Provider>
  );
};

export const useSyncableBounds = (): SyncableBounds => {
  const [ zoomed, setZoomed ] = useState(false);
  const mousePosition = useRef();
  const syncContext = useContext(SyncContext);
  const zoomSetter = syncContext?.setZoomed ?? setZoomed;
  const scaleUpdateDispatcher = syncContext?.dispatchScaleUpdate;
  const fireSignal = syncContext?.fireSignal;
  const dataScaleUpdater = syncContext?.dataScaleUpdater;
  const minMaxRef = syncContext?.minMaxRef;
  const syncRef: MutableRefObject<uPlot.SyncPubSub> = syncContext?.syncRef;
  const setZoomedMax = syncContext?.setZoomedMax;
  const setZoomedMin = syncContext?.setZoomedMin;

  // const contextPresent = syncContext !== undefined

  // if contextPresent
  const boundsOptions = useMemo(() => {
    console.log('being defined');
    return {
      cursor: {
        bind: {
          dblclick: (chart: uPlot, _target: EventTarget, handler: (e: Event) => void) => {
            return (e: Event) => {
              zoomSetter(false);
              if (minMaxRef) {
                console.log('scale min max', minMaxRef.current.max);
                chart.setScale('x', { max: minMaxRef.current.max, min: minMaxRef.current.min });
                chart.setSelect({ height: 0, width: 0 }, false);
                // syncRef.current.pub();
              } else {
                handler(e);
              }
            };
          },
          mousedown: (_uPlot: uPlot, _target: EventTarget, handler: (e: Event) => void) => {
            return (e: MouseEvent) => {
              mousePosition.current = [ e.clientX, e.clientY ];
              handler(e);
            };
          },
          mouseup: (chart: uPlot, _target: EventTarget, handler: (e: Event) => void) => {
            return (e: MouseEvent) => {
              if (!mousePosition.current) {
                handler(e);
                return;
              }
              if (distance(e.clientX, 0, mousePosition.current[0], 0) > 5 && chart.scales.x) {
                const positions = [ e.clientX, mousePosition.current[0] ];
                const maxPos = Math.max(...positions);
                const minPos = Math.min(...positions);
                console.log('ps', minPos, maxPos);
                window.tart = chart;
                const max = chart.posToVal(maxPos, 'x');
                const min = chart.posToVal(minPos, 'x ');
                setZoomedMax?.(max);
                setZoomedMin?.(min);
                zoomSetter(true);
              } else {
                handler(e);
              }
              mousePosition.current = undefined;
            };
          },
        },
        drag: { dist: 5, uni: 10, x: true, y: false },
        sync: syncRef && {
          key: syncRef.current.key,
          scales: [ syncRef.current.key, null ],
          setSeries: false,
        },
      },
      hooks: dataScaleUpdater && {
        init: [ fireSignal ],
        setData: [ dataScaleUpdater ],
        setScale: [ scaleUpdateDispatcher ],
      },
      // scales: { x: { max: minMaxRef.current.max, min: minMaxRef.current.min, time: false } },
    };
  }, [
    zoomSetter,
    dataScaleUpdater,
    syncRef,
    minMaxRef,
    fireSignal,
    scaleUpdateDispatcher,
    setZoomedMax,
    setZoomedMin,
  ]);

  return syncContext ? { ...syncContext, boundsOptions } : { boundsOptions, zoomed };
};

// use sync: use  sync context else setState
