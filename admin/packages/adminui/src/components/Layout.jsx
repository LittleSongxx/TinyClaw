import React, { useEffect, useState } from "react";
import Sidebar from "./Sidebar";
import Header from "./Header";
import { Outlet } from "react-router-dom";
import { useUser } from "../context/UserContext.jsx";

export default function Layout() {
    const [userInfo, setUserInfo] = useState({ username: "" });
    const { user } = useUser();

    useEffect(() => {
        if (user) {
            setUserInfo(user);
        }
    }, [user]);

    return (
        <div className="flex h-screen flex-col overflow-hidden">
            <div className="shrink-0">
                <Header username={userInfo.username} />
            </div>

            <div className="flex min-h-0 flex-1 overflow-hidden">
                <Sidebar />

                <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
                    <div className="h-full min-h-0 w-full overflow-x-hidden overflow-y-auto">
                        <div className="min-h-full">
                            <Outlet />
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
