"use client";

import { useEffect } from "react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("[GlobalError]", error);
  }, [error]);

  return (
    <div className="min-h-screen bg-page-bg flex items-center justify-center px-4">
      <div className="w-full max-w-[440px] bg-white rounded-[10px] p-10 shadow-sm text-center">
        <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-tiger-red mb-4">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
            <path
              d="M12 9v4M12 17h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"
              stroke="white"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>
        <h1 className="text-lg font-semibold text-semi-black mb-2">
          เกิดข้อผิดพลาด / Application Error
        </h1>
        <p className="text-sm text-semi-grey mb-1">
          {error.message || "An unexpected error occurred."}
        </p>
        {error.digest && (
          <p className="text-xs text-semi-grey mb-4 font-mono">
            Digest: {error.digest}
          </p>
        )}
        <button
          onClick={reset}
          className="mt-4 px-6 py-2 rounded-[1000px] bg-tiger-red text-white text-sm font-semibold hover:bg-tiger-red/90 transition-colors"
        >
          ลองใหม่ / Try Again
        </button>
      </div>
    </div>
  );
}
