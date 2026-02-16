// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

// Login SDK for changkun.de services.
// Usage: <script src="https://login.changkun.de/sdk.js"></script>
//
// API:
//   changkunLogin.check()          - Returns Promise<{ok, username}>
//   changkunLogin.login([redirect]) - Redirects to login page
//   changkunLogin.logout([redirect]) - Clears auth cookie and redirects
//   changkunLogin.getToken()       - Returns the auth token or null

(function(global) {
    'use strict';

    var VERIFY_URL = 'https://login.changkun.de/verify';
    var LOGIN_URL  = 'https://login.changkun.de/';

    function getToken() {
        var params = new URLSearchParams(window.location.search);
        var t = params.get('token');
        if (t) return t;

        var match = document.cookie.match(/(?:^|;\s*)auth=([^;]*)/);
        return match ? match[1] : null;
    }

    function check() {
        var token = getToken();
        if (!token) {
            return Promise.resolve({ ok: false, username: '' });
        }

        return fetch(VERIFY_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token: token }),
        })
        .then(function(resp) {
            if (!resp.ok) return { ok: false, username: '' };
            return resp.json().then(function(data) {
                return { ok: true, username: data.username || '' };
            });
        })
        .catch(function() {
            return { ok: false, username: '' };
        });
    }

    function login(redirect) {
        var r = redirect || window.location.href;
        window.location.href = LOGIN_URL + '?redirect=' + encodeURIComponent(r);
    }

    function logout(redirect) {
        document.cookie = 'auth=; Domain=changkun.de; Path=/; Max-Age=0';
        document.cookie = 'auth=; Path=/; Max-Age=0';
        var r = redirect || window.location.origin + window.location.pathname;
        window.location.replace(r);
    }

    global.changkunLogin = {
        check: check,
        login: login,
        logout: logout,
        getToken: getToken,
    };
})(window);
