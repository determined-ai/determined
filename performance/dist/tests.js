/******/ (() => { // webpackBootstrap
/******/ 	"use strict";
/******/ 	// The require scope
/******/ 	var __webpack_require__ = {};
/******/ 	
/************************************************************************/
/******/ 	/* webpack/runtime/compat get default export */
/******/ 	(() => {
/******/ 		// getDefaultExport function for compatibility with non-harmony modules
/******/ 		__webpack_require__.n = (module) => {
/******/ 			var getter = module && module.__esModule ?
/******/ 				() => (module['default']) :
/******/ 				() => (module);
/******/ 			__webpack_require__.d(getter, { a: getter });
/******/ 			return getter;
/******/ 		};
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/define property getters */
/******/ 	(() => {
/******/ 		// define getter functions for harmony exports
/******/ 		__webpack_require__.d = (exports, definition) => {
/******/ 			for(var key in definition) {
/******/ 				if(__webpack_require__.o(definition, key) && !__webpack_require__.o(exports, key)) {
/******/ 					Object.defineProperty(exports, key, { enumerable: true, get: definition[key] });
/******/ 				}
/******/ 			}
/******/ 		};
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/hasOwnProperty shorthand */
/******/ 	(() => {
/******/ 		__webpack_require__.o = (obj, prop) => (Object.prototype.hasOwnProperty.call(obj, prop))
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/make namespace object */
/******/ 	(() => {
/******/ 		// define __esModule on exports
/******/ 		__webpack_require__.r = (exports) => {
/******/ 			if(typeof Symbol !== 'undefined' && Symbol.toStringTag) {
/******/ 				Object.defineProperty(exports, Symbol.toStringTag, { value: 'Module' });
/******/ 			}
/******/ 			Object.defineProperty(exports, '__esModule', { value: true });
/******/ 		};
/******/ 	})();
/******/ 	
/************************************************************************/
var __webpack_exports__ = {};
// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  "default": () => (/* binding */ tests),
  "options": () => (/* binding */ options)
});

;// CONCATENATED MODULE: external "k6"
const external_k6_namespaceObject = require("k6");
;// CONCATENATED MODULE: external "k6/http"
const http_namespaceObject = require("k6/http");
var http_default = /*#__PURE__*/__webpack_require__.n(http_namespaceObject);
;// CONCATENATED MODULE: ./src/tests.ts


var clusterURL = __ENV.DET_MASTER;
var masterEndpoint = '/api/v1/master'; // const userEndpoint = '/api/v1/users'
// const vuIDs = {};
// const userInfo = {};

var scenarios = {
  smoke_test: {
    tags: {
      test_type: 'smoke'
    },
    executor: 'shared-iterations',
    vus: 3,
    maxDuration: "5s",
    iterations: 5
  } // average_load_test: {
  //     tags: { test_type: 'average' },
  //     executor: 'ramping-vus',
  //     stages: [
  //         { duration: '10s', target: 50 },
  //         { duration: '60s', target: 50 },
  //         { duration: '10s', target: 0 }
  //     ],
  //     startTime: "1m"
  // },
  // stress_test: {
  //     tags: { test_type: 'stress' },
  //     executor: 'ramping-vus',
  //     stages: [
  //         { duration: '10s', target: 175 },
  //         { duration: '20s', target: 175 },
  //         { duration: '10s', target: 0 }
  //     ],
  //     startTime: "140s"
  // },
  // soak_test: {
  //     tags: { test_type: 'soak' },
  //     executor: 'ramping-vus',
  //     stages: [
  //         { duration: '5s', target: 50 },
  //         { duration: '1m', target: 50 },
  //         { duration: '1m', target: 0 }
  //     ],
  //     startTime: "180s"
  // },
  // spike_test: {
  //     tags: { test_type: 'spike' },
  //     executor: 'ramping-vus',
  //     stages: [
  //         { duration: '1m', target: 500 },
  //         { duration: '15s', target: 0 },
  //     ],
  //     startTime: "305s"
  // },
  // breakpoint_test: {
  //     tags: { test_type: 'breakpoint' },
  //     executor: 'ramping-arrival-rate',
  //     preAllocatedVUs: 0,
  //     stages: [
  //         { duration: '2m', target: 30000 },
  //     ],
  //     startTime: "380s",
  // },

};
var options = {
  scenarios: scenarios,
  thresholds: {
    'http_req_failed{test_type:breakpoint}': [{
      threshold: 'rate<0.05',
      abortOnFail: true
    }]
  }
}; // test per endpoint per filter set
//   for each test, at least smoke test + average load
// export function setup() {
//     const userRequest = http.get(`${clusterURL}${userEndpoint}`);
//     console.log(`Status is: ${userRequest.status}`);
//     const userIds = Array.from(Array(100).keys())
//     const userInformation = userIds.map((u) => ({
//         id: `u-${u}`,
//         displayName: `user_${u}`,
//     }));
//     userInformation.forEach((u) => vuIDs)
//     userInfo = userInformation;
// }

/* harmony default export */ function tests() {
  // const vuId = vu.idInTest;
  // const userVuInfo = 
  var res = http_default().get("".concat(clusterURL).concat(masterEndpoint));
  (0,external_k6_namespaceObject.check)(res, {
    '200 response': function response(r) {
      return r.status == 200;
    }
  });
}
var __webpack_export_target__ = exports;
for(var i in __webpack_exports__) __webpack_export_target__[i] = __webpack_exports__[i];
if(__webpack_exports__.__esModule) Object.defineProperty(__webpack_export_target__, "__esModule", { value: true });
/******/ })()
;
//# sourceMappingURL=tests.js.map