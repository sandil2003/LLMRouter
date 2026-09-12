import React, { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';

export const Layout: React.FC = () => {
  const [refreshKey, setRefreshKey] = useState(0);

  const triggerRefresh = () => {
    setRefreshKey((k) => k + 1);
  };

  return (
    <div className="app-container">
      <Sidebar />
      <div className="main-wrapper">
        <Header onRefresh={triggerRefresh} />
        <main className="content-area">
          <Outlet context={{ refreshKey }} />
        </main>
      </div>
    </div>
  );
};
