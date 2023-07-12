function ready(fn) {
    if (document.readyState !== 'loading') {
        fn();
    } else {
        document.addEventListener('DOMContentLoaded', fn);
    }
}


ready(function() {
    const errorDivId = "error";
    document.body.addEventListener("htmx:responseError", function(e) {
        document.getElementById(errorDivId).innerHTML = e.detail.xhr.response;
        document.getElementById(errorDivId).classList.remove("dn");
    });
    document.body.addEventListener("htmx:afterOnLoad", function(e) {
        if (e.detail.successful) {
            document.getElementById(errorDivId).innerHTML = "";
            document.getElementById(errorDivId).classList.add("dn");
        }
    });
    document.body.addEventListener("htmx:sendError", function(e) {
        window.location = window.location.protocol + "//" + window.location.host + e.detail.pathInfo.requestPath;
    });
    document.body.addEventListener("htmx:beforeRequest", function(e) {
        document.getElementById("loader").classList.remove("dn");
    });
    document.body.addEventListener("htmx:afterRequest", function(e) {
        document.getElementById("loader").classList.add("dn");
    });
    document.body.addEventListener("htmx:historyRestore", function(e) {
        document.getElementById("loader").classList.add("dn");
    });

    // check whether current browser supports WebAuthn
    if (!window.PublicKeyCredential) {
        alert("Error: this browser does not support WebAuthn");
        return;
    }

})


// Base64 to ArrayBuffer
function bufferDecode(value) {
    value = value.replace(/-/g, '+').replace(/_/g, '/');

    var pad = value.length % 4;
    if(pad) {
        if(pad === 1) {
            throw new Error('InvalidLengthError: Input base64url string is the wrong length to determine padding');
        }
        value += new Array(5-pad).join('=');
    }

    return Uint8Array.from(atob(value), c => c.charCodeAt(0));
}

// ArrayBuffer to URLBase64
function bufferEncode(value) {
    return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
        .replace(/\+/g, "-")
        .replace(/\//g, "_")
        .replace(/=/g, "");
}

function registerUser(username = "") {

    if ($("#username").length) {
        username = $("#username").val()
    }

    $.get(
        '/register/begin/' + username,
        null,
        function (data) {
            return data
        },
        'json')
        .then((credentialCreationOptions) => {
            credentialCreationOptions.publicKey.challenge = bufferDecode(credentialCreationOptions.publicKey.challenge);
            credentialCreationOptions.publicKey.user.id = bufferDecode(credentialCreationOptions.publicKey.user.id);
            if (credentialCreationOptions.publicKey.excludeCredentials) {
                for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
                    credentialCreationOptions.publicKey.excludeCredentials[i].id = bufferDecode(credentialCreationOptions.publicKey.excludeCredentials[i].id);
                }
            }

            return navigator.credentials.create({
                publicKey: credentialCreationOptions.publicKey
            })
        })
        .then((credential) => {
            let attestationObject = credential.response.attestationObject;
            let clientDataJSON = credential.response.clientDataJSON;
            let rawId = credential.rawId;

            $.post(
                '/register/finish/' + username,
                JSON.stringify({
                    id: credential.id,
                    rawId: bufferEncode(rawId),
                    type: credential.type,
                    response: {
                        attestationObject: bufferEncode(attestationObject),
                        clientDataJSON: bufferEncode(clientDataJSON),
                    },
                }),
                function (data) {
                    return data
                },
                'json')
        })
        .then((success) => {
            if (window.location.pathname === "/profile") {
                window.location.reload()
            } else {
                window.location = "/profile"
            }
        })
        .catch((error) => {
            console.error(error)
            $("#error").text(error);
            $("#error").removeClass("dn");
        })
}

function loginUser() {

    username = $("#username").val()
    if (username === "") {
        alert("Please enter a username");
        return;
    }

    $.get(
        '/login/begin/' + username,
        null,
        function (data) {
            return data
        },
        'json')
        .then((credentialRequestOptions) => {
            credentialRequestOptions.publicKey.challenge = bufferDecode(credentialRequestOptions.publicKey.challenge);
            credentialRequestOptions.publicKey.allowCredentials.forEach(function (listItem) {
                listItem.id = bufferDecode(listItem.id)
            });

            return navigator.credentials.get({
                publicKey: credentialRequestOptions.publicKey
            })
        })
        .then((assertion) => {
            let authData = assertion.response.authenticatorData;
            let clientDataJSON = assertion.response.clientDataJSON;
            let rawId = assertion.rawId;
            let sig = assertion.response.signature;
            let userHandle = assertion.response.userHandle;

            $.post(
                '/login/finish/' + username,
                JSON.stringify({
                    id: assertion.id,
                    rawId: bufferEncode(rawId),
                    type: assertion.type,
                    response: {
                        authenticatorData: bufferEncode(authData),
                        clientDataJSON: bufferEncode(clientDataJSON),
                        signature: bufferEncode(sig),
                        userHandle: bufferEncode(userHandle),
                    },
                }),
                function (data) {
                    return data
                },
                'json')
        })
        .then((success) => {
            window.location = "/profile"
        })
        .catch((error) => {
            console.error(error)
            $("#error").text(error);
            $("#error").removeClass("dn");
        })
}