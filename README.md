<p align="center"><a href="https://www.twitch.tv/" target="_blank" rel="noopener noreferrer"><img src="doc/Combo_Purple_RGB.png" height="196" alt="Twitch TV logo"></a></p>

# TUT, Twitch Unfollow Tracker

A prototype twitch unfollow tracker utlizing the latest twitch API and developed using Go.

<p align="center"><img src="doc/tut.gif" alt="TUT demo"></p>

_Please Noted: The GIF may not reflect the latest build. And the followers update interval in the demo GIF is 1 minute for demonstration purpose and I also have fast-forwarded the GIF. The default update interval is 60 minutes._

## [中文說明](https://github.com/StinkingBanana/tut/wiki/Chinese-%E4%B8%AD%E6%96%87%E8%AA%AA%E6%98%8E)

# Installation

## Install via go command:

```
$ go get -u github.com/stinkingbanana/tut
```

## Uninstall via go command:

```
$ go clean -i github.com/stinkingbanana/tut
```

## Download from realease page:

[https://github.com/StinkingBanana/tut/releases](https://github.com/StinkingBanana/tut/releases)

## To Run from source

```
$ go run .
```

# To Use

1. Start the program
2. Enter the clientID of yours or use the default clientID. (Your input will be remembered.)  
   Obtain your own clientID from [https://dev.twitch.tv/dashboard](https://dev.twitch.tv/dashboard).
3. Enter the Access token of yours. (Your input will be remembered.)

   - Obtain your Access Token using the following URL

   ```
     https://id.twitch.tv/oauth2/authorize?response_type=token&client_id={ClientID}&redirect_uri=http://localhost&scope=viewing_activity_read
   ```

   - Replace {ClientID} with your ClientID and enter in your browser.
   - Authorize it.
   - If nothing goes wrong, you should get an empty white page with URL similar to http://localhost/#access_token={AccessToken}&...
   - Extract that {AccessToken} and that's what we need for this step.
   - Note: the access token will expire after sometime. If the access token no longer works, generate a new one using the same method.

4. Enter twitch username to track. (Your input will be remembered.)
5. Enter update interval. Default is 60 minutes. (Your input will be remembered.)
6. Enter the server port or use the default port. (Your input will be remembered.)
7. DONE. You just keep the program alive, it will monitor unfollows and refollows.

# Available Endpoints

The program will host a server at `http://localhost:5487`.

<p align="center"><img src="doc/getunfollowers.jpg" alt="TUT endpoints demo"></p>

## Get Followers

```

http://localhost:5487/followers

```

## Get Unfollowers

```

http://localhost:5487/unfollowers

```

## More endpoints?

Please check

```

http://localhost:5487

```

# NOTE

- Please make sure you sync or keep your computer time updated.
- This is a quick and dirty prototype, not perfect at all. Let me know if there is any issues.
- The default database name is `TUT.db`, it will be created wherever you run the program.
  So technially you could run multiple instance of the program, but you need to make sure to run it under different directory and using different server port.
  Oh, and don't forget to use different ClientID or OAuth token, so you don't overload your API limit.
- Please be paitent if you have large amount of followers.
  Due to API request limit, it can only process 3000 followers' ID per minute or 30 followers detailed profile info per minute.
  If you have provided valid oauth token, it will process 12000 followers' ID per minute or 120 followers detailed profile info per minute.

# FAQ

## How to Obtain Client ID?

Obtain your own clientID from [https://dev.twitch.tv/dashboard](https://dev.twitch.tv/dashboard)
